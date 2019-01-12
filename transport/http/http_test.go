package http

import (
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/micro/go-micro/transport"
)

func call(b *testing.B, c int) {
	b.StopTimer()

	tr := NewTransport()

	// server listen
	l, err := tr.Listen("localhost:0")
	if err != nil {
		b.Fatal(err)
	}
	defer l.Close()

	// socket func
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

	// accept connections
	go func() {
		if err := l.Accept(fn); err != nil {
			select {
			case <-done:
			default:
				b.Fatalf("Unexpected accept err: %v", err)
			}
		}
	}()

	m := transport.Message{
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"message": "Hello World"}`),
	}

	// client connection
	client, err := tr.Dial(l.Addr())
	if err != nil {
		b.Fatalf("Unexpected dial err: %v", err)
	}

	send := func(c transport.Client) {
		// send message
		if err := c.Send(&m); err != nil {
			b.Fatalf("Unexpected send err: %v", err)
		}

		var rm transport.Message
		// receive message
		if err := c.Recv(&rm); err != nil {
			b.Fatalf("Unexpected recv err: %v", err)
		}
	}

	// warm
	for i := 0; i < 10; i++ {
		send(client)
	}

	client.Close()

	ch := make(chan int, c*4)

	var wg sync.WaitGroup
	wg.Add(c)

	for i := 0; i < c; i++ {
		go func() {
			cl, err := tr.Dial(l.Addr())
			if err != nil {
				b.Fatalf("Unexpected dial err: %v", err)
			}
			defer cl.Close()

			for range ch {
				send(cl)
			}

			wg.Done()
		}()
	}

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		ch <- i
	}

	b.StopTimer()
	close(ch)

	wg.Wait()

	// finish
	close(done)
}

func BenchmarkTransport1(b *testing.B) {
	call(b, 1)
}

func BenchmarkTransport8(b *testing.B) {
	call(b, 8)
}

func BenchmarkTransport16(b *testing.B) {
	call(b, 16)
}

func BenchmarkTransport64(b *testing.B) {
	call(b, 64)
}

func BenchmarkTransport128(b *testing.B) {
	call(b, 128)
}

func expectedPort(t *testing.T, expected string, lsn transport.Listener) {
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

	lsn, err := tp.Listen("127.0.0.1:0")
	if err != nil {
		t.Errorf("Did not expect an error, got %s", err)
	}

	lsn.Close()
	lsn1.Close()
	lsn2.Close()
}

func TestHTTPTransportCommunication(t *testing.T) {
	tr := NewTransport()

	l, err := tr.Listen("127.0.0.1:0")
	if err != nil {
		t.Errorf("Unexpected listen err: %v", err)
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
				t.Errorf("Unexpected accept err: %v", err)
			}
		}
	}()

	c, err := tr.Dial(l.Addr())
	if err != nil {
		t.Errorf("Unexpected dial err: %v", err)
	}
	defer c.Close()

	m := transport.Message{
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"message": "Hello World"}`),
	}

	if err := c.Send(&m); err != nil {
		t.Errorf("Unexpected send err: %v", err)
	}

	var rm transport.Message

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

	l, err := tr.Listen("127.0.0.1:0")
	if err != nil {
		t.Errorf("Unexpected listen err: %v", err)
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

			sock.(*httpSocket).error(&transport.Message{
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

	m := transport.Message{
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"message": "Hello World"}`),
	}

	if err := c.Send(&m); err != nil {
		t.Errorf("Unexpected send err: %v", err)
	}

	var rm transport.Message

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
	tr := NewTransport(transport.Timeout(time.Millisecond * 100))

	l, err := tr.Listen("127.0.0.1:0")
	if err != nil {
		t.Errorf("Unexpected listen err: %v", err)
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
				t.Errorf("Unexpected accept err: %v", err)
			}
		}
	}()

	c, err := tr.Dial(l.Addr())
	if err != nil {
		t.Errorf("Unexpected dial err: %v", err)
	}
	defer c.Close()

	m := transport.Message{
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
