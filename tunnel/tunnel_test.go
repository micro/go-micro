package tunnel

import (
	"testing"

	"github.com/micro/go-micro/transport"
)

// testAccept will accept connections on the transport, create a new link and tunnel on top
func testAccept(t *testing.T, tun Tunnel, wait chan bool) {
	// listen on some virtual address
	tl, err := tun.Listen("test-tunnel")
	if err != nil {
		t.Fatal(err)
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
}

// testSend will create a new link to an address and then a tunnel on top
func testSend(t *testing.T, tun Tunnel) {
	// dial a new session
	c, err := tun.Dial("test-tunnel")
	if err != nil {
		t.Fatal(err)
	}
	//defer c.Close()

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
	tun := NewTunnel(Nodes(":9096"))
	err := tun.Connect()
	if err != nil {
		t.Fatal(err)
	}
	//defer tun.Close()

	wait := make(chan bool)

	// start accepting connections
	go testAccept(t, tun, wait)

	// send a message
	testSend(t, tun)

	// wait until message is received
	<-wait
}
