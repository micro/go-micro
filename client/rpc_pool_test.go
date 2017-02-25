package client

import (
	"testing"
	"time"

	"github.com/micro/go-micro/transport"
	"github.com/micro/go-micro/transport/mock"
)

func testPool(t *testing.T, size int, ttl time.Duration) {
	// zero pool
	p := newPool(size, ttl)

	// mock transport
	tr := mock.NewTransport()

	// listen
	l, err := tr.Listen(":0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	// accept loop
	go func() {
		for {
			if err := l.Accept(func(s transport.Socket) {
				for {
					var msg transport.Message
					if err := s.Recv(&msg); err != nil {
						return
					}
					if err := s.Send(&msg); err != nil {
						return
					}
				}
			}); err != nil {
				return
			}
		}
	}()

	for i := 0; i < 10; i++ {
		// get a conn
		c, err := p.getConn(l.Addr(), tr)
		if err != nil {
			t.Fatal(err)
		}

		msg := &transport.Message{
			Body: []byte(`hello world`),
		}

		if err := c.Send(msg); err != nil {
			t.Fatal(err)
		}

		var rcv transport.Message

		if err := c.Recv(&rcv); err != nil {
			t.Fatal(err)
		}

		if string(rcv.Body) != string(msg.Body) {
			t.Fatalf("got %v, expected %v", rcv.Body, msg.Body)
		}

		// release the conn
		p.release(l.Addr(), c, nil)

		p.Lock()
		if i := len(p.conns[l.Addr()]); i > size {
			p.Unlock()
			t.Fatalf("pool size %d is greater than expected %d", i, size)
		}
		p.Unlock()
	}
}

func TestRPCPool(t *testing.T) {
	testPool(t, 0, time.Minute)
	testPool(t, 2, time.Minute)
}
