package tunnel

import (
	"sync"
	"testing"
	"time"

	"github.com/micro/go-micro/transport"
)

// testAccept will accept connections on the transport, create a new link and tunnel on top
func testAccept(t *testing.T, tun Tunnel, wg *sync.WaitGroup) {
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
		wg.Done()
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
	defer c.Close()

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
	tun := NewTunnel(Nodes("127.0.0.1:9096"))
	err := tun.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer tun.Close()

	var wg sync.WaitGroup

	// start accepting connections
	wg.Add(1)
	go testAccept(t, tun, &wg)

	// send a message
	testSend(t, tun)

	// wait until message is received
	wg.Wait()
}

func TestTwoTunnel(t *testing.T) {
	// create a new tunnel client
	tunA := NewTunnel(
		Address("127.0.0.1:9096"),
		Nodes("127.0.0.1:9097"),
	)

	// create a new tunnel server
	tunB := NewTunnel(
		Address("127.0.0.1:9097"),
	)

	// start tunB
	err := tunB.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer tunB.Close()

	time.Sleep(time.Millisecond * 50)

	// start tunA
	err = tunA.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer tunA.Close()

	time.Sleep(time.Millisecond * 50)

	var wg sync.WaitGroup

	// start accepting connections
	// on tunnel A
	wg.Add(1)
	go testAccept(t, tunA, &wg)

	time.Sleep(time.Millisecond * 50)

	// dial and send via B
	testSend(t, tunB)

	// wait until done
	wg.Wait()
}
