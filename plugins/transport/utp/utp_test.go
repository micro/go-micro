package utp

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/asim/go-micro/v3/transport"
)

func expectedPort(t *testing.T, expected string, lsn transport.Listener) {
	parts := strings.Split(lsn.Addr(), ":")
	port := parts[len(parts)-1]

	if port != expected {
		lsn.Close()
		t.Fatalf("Expected address to be `%s`, got `%s`", expected, port)
	}
}

func testUTPTransport(t *testing.T, secure bool) {
	tr := NewTransport(transport.Secure(secure))

	l, err := tr.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Unexpected listen err: %v", err)
	}
	defer l.Close()

	fn := func(sock transport.Socket) {
		defer sock.Close()

		for {
			var m transport.Message
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
				t.Fatalf("Unexpected accept err: %v", err)
			}
		}
	}()

	c, err := tr.Dial(l.Addr())
	if err != nil {
		t.Fatalf("Unexpected dial err: %v", err)
	}
	defer c.Close()

	m := transport.Message{
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"message": "Hello World"}`),
	}

	if err := c.Send(&m); err != nil {
		t.Fatalf("Unexpected send err: %v", err)
	}

	var rm transport.Message

	if err := c.Recv(&rm); err != nil {
		t.Fatalf("Unexpected recv err: %v", err)
	}

	if string(rm.Body) != string(m.Body) {
		t.Fatalf("Expected %v, got %v", m.Body, rm.Body)
	}

	close(done)
}

func TestUTPTransportPortRange(t *testing.T) {
	tp := NewTransport()

	lsn1, err := tp.Listen(":44444-44448")
	if err != nil {
		t.Fatalf("Did not expect an error, got %s", err)
	}
	expectedPort(t, "44444", lsn1)

	lsn2, err := tp.Listen(":44444-44448")
	if err != nil {
		t.Fatalf("Did not expect an error, got %s", err)
	}
	expectedPort(t, "44445", lsn2)

	lsn, err := tp.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Did not expect an error, got %s", err)
	}

	lsn.Close()
	lsn1.Close()
	lsn2.Close()
}

func TestUTPTransportError(t *testing.T) {
	tr := NewTransport()

	l, err := tr.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Unexpected listen err: %v", err)
	}
	defer l.Close()

	fn := func(sock transport.Socket) {
		defer sock.Close()

		for {
			var m transport.Message
			if err := sock.Recv(&m); err != nil {
				if err == io.EOF {
					return
				}
				t.Fatal(err)
			}
		}
	}

	done := make(chan bool)

	go func() {
		if err := l.Accept(fn); err != nil {
			select {
			case <-done:
			default:
				t.Fatalf("Unexpected accept err: %v", err)
			}
		}
	}()

	c, err := tr.Dial(l.Addr())
	if err != nil {
		t.Fatalf("Unexpected dial err: %v", err)
	}
	defer c.Close()

	m := transport.Message{
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"message": "Hello World"}`),
	}

	if err := c.Send(&m); err != nil {
		t.Fatalf("Unexpected send err: %v", err)
	}

	close(done)
}

func TestUTPTransportTimeout(t *testing.T) {
	tr := NewTransport(transport.Timeout(time.Millisecond * 100))

	l, err := tr.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Unexpected listen err: %v", err)
	}
	defer l.Close()

	done := make(chan bool)

	fn := func(sock transport.Socket) {
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
			var m transport.Message

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
				t.Fatalf("Unexpected accept err: %v", err)
			}
		}
	}()

	c, err := tr.Dial(l.Addr())
	if err != nil {
		t.Fatalf("Unexpected dial err: %v", err)
	}
	defer c.Close()

	m := transport.Message{
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"message": "Hello World"}`),
	}

	if err := c.Send(&m); err != nil {
		t.Fatalf("Unexpected send err: %v", err)
	}

	<-done
}

func TestUTPTransportCommunication(t *testing.T) {
	testUTPTransport(t, false)
}

func TestUTPTransportTLSCommunication(t *testing.T) {
	testUTPTransport(t, true)
}
