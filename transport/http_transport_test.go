package transport

import (
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

func expectedPort(t *testing.T, expected string, lsn Listener) {
	_, port, err := net.SplitHostPort(lsn.Addr())
	if err != nil {
		t.Errorf("Expected address to be `%s`, got error: %v", expected, err)
	}

	if port != expected {
		lsn.Close()
		t.Errorf("Expected address to be `%s`, got `%s`", expected, port)
	}
}

func TestHTTPTransportPortRange(t *testing.T) {
	tp := NewHTTPTransport()

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

	lsn, err := tp.Listen("127.0.0.1:0")
	if err != nil {
		t.Errorf("Did not expect an error, got %s", err)
	}

	lsn.Close()
	lsn1.Close()
	lsn2.Close()
}

func TestHTTPTransportCommunication(t *testing.T) {
	tr := NewHTTPTransport()

	l, err := tr.Listen("127.0.0.1:0")
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
	tr := NewHTTPTransport()

	l, err := tr.Listen("127.0.0.1:0")
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
	tr := NewHTTPTransport(Timeout(time.Millisecond * 100))

	l, err := tr.Listen("127.0.0.1:0")
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

func TestHTTPTransportCloseWhenRecv(t *testing.T) {
	tr := NewHTTPTransport()

	l, err := tr.Listen("127.0.0.1:0")
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
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			var rm Message

			if err := c.Recv(&rm); err != nil {
				if err == io.EOF {
					return
				}
			}
		}
	}()
	for i := 1; i < 3; i++ {
		if err := c.Send(&m); err != nil {
			t.Errorf("Unexpected send err: %v", err)
		}
	}
	close(done)

	c.Close()
	wg.Wait()
}

func TestHTTPTransportMultipleSendWhenRecv(t *testing.T) {
	tr := NewHTTPTransport()

	l, err := tr.Listen("127.0.0.1:0")
	if err != nil {
		t.Errorf("Unexpected listen err: %v", err)
	}
	defer l.Close()

	readyToSend := make(chan struct{})
	m := Message{
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"message": "Hello World"}`),
	}

	wgSend := sync.WaitGroup{}
	fn := func(sock Socket) {
		defer sock.Close()

		for {
			var mr Message
			if err := sock.Recv(&mr); err != nil {
				return
			}
			wgSend.Add(1)
			go func() {
				defer wgSend.Done()
				<-readyToSend
				if err := sock.Send(&m); err != nil {
					return
				}
			}()
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

	c, err := tr.Dial(l.Addr(), WithStream())
	if err != nil {
		t.Errorf("Unexpected dial err: %v", err)
	}
	defer c.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	readyForRecv := make(chan struct{})
	go func() {
		defer wg.Done()
		close(readyForRecv)
		for {
			var rm Message
			if err := c.Recv(&rm); err != nil {
				if err == io.EOF {
					return
				}
			}
		}
	}()
	<-readyForRecv
	for i := 0; i < 3; i++ {
		if err := c.Send(&m); err != nil {
			t.Errorf("Unexpected send err: %v", err)
		}
	}
	close(readyToSend)
	wgSend.Wait()
	close(done)

	c.Close()
	wg.Wait()
}
