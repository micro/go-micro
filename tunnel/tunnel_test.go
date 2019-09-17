package tunnel

import (
	"sync"
	"testing"
	"time"

	"github.com/micro/go-micro/transport"
)

// testAccept will accept connections on the transport, create a new link and tunnel on top
func testAccept(t *testing.T, tun Tunnel, wait chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	// listen on some virtual address
	tl, err := tun.Listen("test-tunnel")
	if err != nil {
		t.Fatal(err)
	}

	// receiver ready; notify sender
	wait <- true

	// accept a connection
	c, err := tl.Accept()
	if err != nil {
		t.Fatal(err)
	}

	// get a message
	for {
		// accept the message
		m := new(transport.Message)
		if err := c.Recv(m); err != nil {
			t.Fatal(err)
		}

		if v := m.Header["test"]; v != "send" {
			t.Fatalf("Accept side expected test:send header. Received: %s", v)
		}

		// now respond
		m.Header["test"] = "accept"
		if err := c.Send(m); err != nil {
			t.Fatal(err)
		}

		wait <- true

		return
	}
}

// testSend will create a new link to an address and then a tunnel on top
func testSend(t *testing.T, tun Tunnel, wait chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	// wait for the listener to get ready
	<-wait

	// dial a new session
	c, err := tun.Dial("test-tunnel")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	m := transport.Message{
		Header: map[string]string{
			"test": "send",
		},
	}

	// send the message
	if err := c.Send(&m); err != nil {
		t.Fatal(err)
	}

	// now wait for the response
	mr := new(transport.Message)
	if err := c.Recv(mr); err != nil {
		t.Fatal(err)
	}

	<-wait

	if v := mr.Header["test"]; v != "accept" {
		t.Fatalf("Message not received from accepted side. Received: %s", v)
	}
}

func TestTunnel(t *testing.T) {
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

	// start tunA
	err = tunA.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer tunA.Close()

	wait := make(chan bool)

	var wg sync.WaitGroup

	wg.Add(1)
	// start the listener
	go testAccept(t, tunB, wait, &wg)

	wg.Add(1)
	// start the client
	go testSend(t, tunA, wait, &wg)

	// wait until done
	wg.Wait()
}

func TestLoopbackTunnel(t *testing.T) {
	// create a new tunnel
	tun := NewTunnel(
		Address("127.0.0.1:9096"),
		Nodes("127.0.0.1:9096"),
	)

	// start tunnel
	err := tun.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer tun.Close()

	time.Sleep(500 * time.Millisecond)

	wait := make(chan bool)

	var wg sync.WaitGroup

	wg.Add(1)
	// start the listener
	go testAccept(t, tun, wait, &wg)

	wg.Add(1)
	// start the client
	go testSend(t, tun, wait, &wg)

	// wait until done
	wg.Wait()
}

func testBrokenTunAccept(t *testing.T, tun Tunnel, wait chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	// listen on some virtual address
	tl, err := tun.Listen("test-tunnel")
	if err != nil {
		t.Fatal(err)
	}

	// receiver ready; notify sender
	wait <- true

	// accept a connection
	c, err := tl.Accept()
	if err != nil {
		t.Fatal(err)
	}

	// accept the message and close the tunnel
	// we do this to simulate loss of network connection
	m := new(transport.Message)
	if err := c.Recv(m); err != nil {
		t.Fatal(err)
	}

	// close all the links
	for _, link := range tun.Links() {
		link.Close()
	}

	// receiver ready; notify sender
	wait <- true

	// accept the message
	m = new(transport.Message)
	if err := c.Recv(m); err != nil {
		t.Fatal(err)
	}

	// notify sender we have received the message
	<-wait
}

func testBrokenTunSend(t *testing.T, tun Tunnel, wait chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	// wait for the listener to get ready
	<-wait

	// dial a new session
	c, err := tun.Dial("test-tunnel")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	m := transport.Message{
		Header: map[string]string{
			"test": "send",
		},
	}

	// send the message
	if err := c.Send(&m); err != nil {
		t.Fatal(err)
	}

	// wait for the listener to get ready
	<-wait

	// give it time to reconnect
	time.Sleep(2 * ReconnectTime)

	// send the message
	if err := c.Send(&m); err != nil {
		t.Fatal(err)
	}

	// wait for the listener to receive the message
	// c.Send merely enqueues the message to the link send queue and returns
	// in order to verify it was received we wait for the listener to tell us
	wait <- true
}

func TestReconnectTunnel(t *testing.T) {
	// create a new tunnel client
	tunA := NewTunnel(
		Address("127.0.0.1:9096"),
		Nodes("127.0.0.1:9097"),
	)

	// create a new tunnel server
	tunB := NewTunnel(
		Address("127.0.0.1:9097"),
	)

	// start tunnel
	err := tunB.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer tunB.Close()

	// we manually override the tunnel.ReconnectTime value here
	// this is so that we make the reconnects faster than the default 5s
	ReconnectTime = 200 * time.Millisecond

	// start tunnel
	err = tunA.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer tunA.Close()

	wait := make(chan bool)

	var wg sync.WaitGroup

	wg.Add(1)
	// start tunnel listener
	go testBrokenTunAccept(t, tunB, wait, &wg)

	wg.Add(1)
	// start tunnel sender
	go testBrokenTunSend(t, tunA, wait, &wg)

	// wait until done
	wg.Wait()
}
