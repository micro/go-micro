package socket

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/asim/nitro/app/network"
)

func expectedPort(t *testing.T, expected string, lsn network.Listener) {
	parts := strings.Split(lsn.Addr(), ":")
	port := parts[len(parts)-1]

	if port != expected {
		lsn.Close()
		t.Errorf("Expected address to be `%s`, got `%s`", expected, port)
	}
}

func TestSocketTransportPortRange(t *testing.T) {
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

func TestSocketTransportCommunication(t *testing.T) {
	tr := NewTransport()

	l, err := tr.Listen(":0")
	if err != nil {
		t.Errorf("Unexpected listen err: %v", err)
	}
	defer l.Close()

	fn := func(sock network.Socket) {
		defer sock.Close()

		for {
			var m network.Message
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

	m := network.Message{
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"message": "Hello World"}`),
	}

	if err := c.Send(&m); err != nil {
		t.Errorf("Unexpected send err: %v", err)
	}

	var rm network.Message

	if err := c.Recv(&rm); err != nil {
		t.Errorf("Unexpected recv err: %v", err)
	}

	if string(rm.Body) != string(m.Body) {
		t.Errorf("Expected %v, got %v", m.Body, rm.Body)
	}

	close(done)
}

func TestSocketTransportError(t *testing.T) {
	tr := NewTransport()

	l, err := tr.Listen(":0")
	if err != nil {
		t.Errorf("Unexpected listen err: %v", err)
	}
	defer l.Close()

	fn := func(sock network.Socket) {
		defer sock.Close()

		for {
			var m network.Message
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
				t.Errorf("Unexpected accept err: %v", err)
			}
		}
	}()

	c, err := tr.Dial(l.Addr())
	if err != nil {
		t.Errorf("Unexpected dial err: %v", err)
	}
	defer c.Close()

	m := network.Message{
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"message": "Hello World"}`),
	}

	if err := c.Send(&m); err != nil {
		t.Errorf("Unexpected send err: %v", err)
	}

	close(done)
}

func TestSocketTransportTimeout(t *testing.T) {
	tr := NewTransport(network.Timeout(time.Millisecond * 100))

	l, err := tr.Listen(":0")
	if err != nil {
		t.Errorf("Unexpected listen err: %v", err)
	}
	defer l.Close()

	done := make(chan bool)

	fn := func(sock network.Socket) {
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
			var m network.Message

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

	m := network.Message{
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
