package transport

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func expectedPort(t *testing.T, expected string, lsn Listener) {
	parts := strings.Split(lsn.Addr(), ":")
	port := parts[len(parts)-1]

	if port != expected {
		lsn.Close()
		t.Errorf("Expected address to be `%s`, got `%s`", expected, port)
	}
}

func TestHTTPTransportPortRange(t *testing.T) {
	tp := NewTransport()

	lsn1, err := tp.Listen(":44444-44448")
	if err != nil {
		t.Errorf("Did not expect an error, got %s", err)
	}
	expectedPort(t, "44444", lsn1)

	lsn2, err := tp.Listen(":44444-44448")
	if err != nil {
		t.Errorf("Did not expect an error, got %s", err)
	}
	expectedPort(t, "44445", lsn2)

	lsn, err := tp.Listen(":0")
	if err != nil {
		t.Errorf("Did not expect an error, got %s", err)
	}

	lsn.Close()
	lsn1.Close()
	lsn2.Close()
}

func TestHTTPTransportCommunication(t *testing.T) {
	tr := NewTransport()

	l, err := tr.Listen(":0")
	if err != nil {
		t.Errorf("Unexpected listen err: %v", err)
	}
	defer l.Close()

	fn := func(sock Socket) {
		defer sock.Close()

		for {
			var m Message
			if err := sock.Recv(&m); err != nil {
				return
			}

			if err := sock.Send(&m); err != nil {
				return
			}
		}
	}

	done := make(chan bool)

	go func() {
		if err := l.Accept(fn); err != nil {
			select {
			case <-done:
			default:
				t.Errorf("Unexpected accept err: %v", err)
			}
		}
	}()

	c, err := tr.Dial(l.Addr())
	if err != nil {
		t.Errorf("Unexpected dial err: %v", err)
	}
	defer c.Close()

	m := Message{
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"message": "Hello World"}`),
	}

	if err := c.Send(&m); err != nil {
		t.Errorf("Unexpected send err: %v", err)
	}

	var rm Message

	if err := c.Recv(&rm); err != nil {
		t.Errorf("Unexpected recv err: %v", err)
	}

	if string(rm.Body) != string(m.Body) {
		t.Errorf("Expected %v, got %v", m.Body, rm.Body)
	}

	close(done)
}

func TestHTTPTransportError(t *testing.T) {
	tr := NewTransport()

	l, err := tr.Listen(":0")
	if err != nil {
		t.Errorf("Unexpected listen err: %v", err)
	}
	defer l.Close()

	fn := func(sock Socket) {
		defer sock.Close()

		for {
			var m Message
			if err := sock.Recv(&m); err != nil {
				if err == io.EOF {
					return
				}
				t.Fatal(err)
			}

			sock.(*httpTransportSocket).error(&Message{
				Body: []byte(`an error occurred`),
			})
		}
	}

	done := make(chan bool)

	go func() {
		if err := l.Accept(fn); err != nil {
			select {
			case <-done:
			default:
				t.Errorf("Unexpected accept err: %v", err)
			}
		}
	}()

	c, err := tr.Dial(l.Addr())
	if err != nil {
		t.Errorf("Unexpected dial err: %v", err)
	}
	defer c.Close()

	m := Message{
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"message": "Hello World"}`),
	}

	if err := c.Send(&m); err != nil {
		t.Errorf("Unexpected send err: %v", err)
	}

	var rm Message

	err = c.Recv(&rm)
	if err == nil {
		t.Fatal("Expected error but got nil")
	}

	if err.Error() != "500 Internal Server Error: an error occurred" {
		t.Fatalf("Did not receive expected error, got: %v", err)
	}

	close(done)
}

func TestHTTPTransportTimeout(t *testing.T) {
	tr := NewTransport(Timeout(time.Millisecond * 100))

	l, err := tr.Listen(":0")
	if err != nil {
		t.Errorf("Unexpected listen err: %v", err)
	}
	defer l.Close()

	done := make(chan bool)

	fn := func(sock Socket) {
		defer func() {
			sock.Close()
			close(done)
		}()

		go func() {
			select {
			case <-done:
				return
			case <-time.After(time.Second):
				t.Fatal("deadline not executed")
			}
		}()

		for {
			var m Message

			if err := sock.Recv(&m); err != nil {
				return
			}
		}
	}

	go func() {
		if err := l.Accept(fn); err != nil {
			select {
			case <-done:
			default:
				t.Errorf("Unexpected accept err: %v", err)
			}
		}
	}()

	c, err := tr.Dial(l.Addr())
	if err != nil {
		t.Errorf("Unexpected dial err: %v", err)
	}
	defer c.Close()

	m := Message{
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"message": "Hello World"}`),
	}

	if err := c.Send(&m); err != nil {
		t.Errorf("Unexpected send err: %v", err)
	}

	<-done
}

type mockListener struct {
}

func (b *mockListener) Accept() (net.Conn, error) {
	server, _ := net.Pipe()
	return server, nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (b *mockListener) Close() error {
	return nil
}

// Addr returns the listener's network address.
func (b *mockListener) Addr() net.Addr {
	return nil
}

func TestOnPanic(t *testing.T) {

	backChan := make(chan interface{})
	err := errors.New("I am random panic thrown")
	h := &httpTransportListener{
		listener: &mockListener{},
		ht: newHTTPTransport(OnPanic(func(obj interface{}) {
			backChan <- err
		})),
	}
	fn := func(Socket) {
		panic(err)
	}
	go h.Accept(fn)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	var recovered interface{}
	select {
	case recovered = <-backChan:
		break
	case <-ctx.Done():
		t.Fatal("Timeout when waiting for panic")
	}
	if err != recovered {
		t.Fatalf("Expected <%#v> but got <%#v>", err, recovered)
	}
}
