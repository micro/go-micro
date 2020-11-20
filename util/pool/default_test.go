package pool

import (
	"testing"
	"time"

	"github.com/asim/nitro/app/network"
	"github.com/asim/nitro/app/network/memory"
)

func testPool(t *testing.T, size int, ttl time.Duration) {
	// mock network
	tr := memory.NewTransport()

	options := Options{
		TTL:       ttl,
		Size:      size,
		Transport: tr,
	}
	// zero pool
	p := newPool(options)

	// listen
	l, err := tr.Listen(":0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	// accept loop
	go func() {
		for {
			if err := l.Accept(func(s network.Socket) {
				for {
					var msg network.Message
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
		c, err := p.Get(l.Addr())
		if err != nil {
			t.Fatal(err)
		}

		msg := &network.Message{
			Body: []byte(`hello world`),
		}

		if err := c.Send(msg); err != nil {
			t.Fatal(err)
		}

		var rcv network.Message

		if err := c.Recv(&rcv); err != nil {
			t.Fatal(err)
		}

		if string(rcv.Body) != string(msg.Body) {
			t.Fatalf("got %v, expected %v", rcv.Body, msg.Body)
		}

		// release the conn
		p.Release(c, nil)

		p.Lock()
		if i := len(p.conns[l.Addr()]); i > size {
			p.Unlock()
			t.Fatalf("pool size %d is greater than expected %d", i, size)
		}
		p.Unlock()
	}
}

func TestClientPool(t *testing.T) {
	testPool(t, 0, time.Minute)
	testPool(t, 2, time.Minute)
}
