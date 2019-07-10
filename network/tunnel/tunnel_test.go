package tunnel

import (
	"testing"

	"github.com/micro/go-micro/network/link"
	"github.com/micro/go-micro/transport"
)

func testAccept(t *testing.T, l transport.Listener, wait chan bool) error {
	// accept new connections on the transport
	// establish a link and tunnel
	return l.Accept(func(s transport.Socket) {
		// convert the socket into a link
		li := link.NewLink(
			link.Socket(s),
		)

		// connect the link e.g start internal buffers
		if err := li.Connect(); err != nil {
			t.Fatal(err)
		}

		// create a new tunnel
		tun := NewTunnel(li)

		// connect the tunnel
		if err := tun.Connect(); err != nil {
			t.Fatal(err)
		}

		// listen on some virtual address
		tl, err := tun.Listen("test-tunnel")
		if err != nil {
			t.Fatal(err)
			return
		}

		// accept a connection
		c, err := tl.Accept()
		if err != nil {
			t.Fatal(err)
		}

		// get a message
		for {
			m := new(transport.Message)
			if err := c.Recv(m); err != nil {
				t.Fatal(err)
			}
			close(wait)
			return
		}
	})
}

func testSend(t *testing.T, addr string) {
	// create a new link
	l := link.NewLink(
		link.Address(addr),
	)

	// connect the link, this includes dialing
	if err := l.Connect(); err != nil {
		t.Fatal(err)
	}

	// create a tunnel on the link
	tun := NewTunnel(l)

	// connect the tunnel with the remote side
	if err := tun.Connect(); err != nil {
		t.Fatal(err)
	}

	// dial a new session
	c, err := tun.Dial("test-tunnel")
	if err != nil {
		t.Fatal(err)
	}

	m := transport.Message{
		Header: map[string]string{
			"test": "header",
		},
	}
	if err := c.Send(&m); err != nil {
		t.Fatal(err)
	}
}

func TestTunnel(t *testing.T) {
	// create a new listener
	tr := transport.NewTransport()
	l, err := tr.Listen(":0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	wait := make(chan bool)

	// start accepting connections
	go testAccept(t, l, wait)

	// send a message
	testSend(t, l.Addr())

	// wait until message is received
	<-wait
}
