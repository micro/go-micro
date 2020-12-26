package main

import (
	"sync"
	"time"

	"github.com/micro/go-micro/v2/transport"
	"github.com/micro/go-micro/v2/tunnel"
	"github.com/micro/go-micro/v2/util/log"
)

// testAccept will accept connections on the transport, create a new link and tunnel on top
func testAccept(tun tunnel.Tunnel, wg *sync.WaitGroup) {
	// listen on some virtual address
	tl, err := tun.Listen("test-tunnel")
	if err != nil {
		log.Fatal(err)
	}

	log.Log("Listening on ", tun.Address())
	wg.Done()

	// accept a connection
	c, err := tl.Accept()
	if err != nil {
		log.Fatal(err)
	}
	log.Log("Accepting connection")

	// get a message
	for {
		m := new(transport.Message)
		if err := c.Recv(m); err != nil {
			log.Fatal(err)
		}
		log.Log("Received message")
		wg.Done()
		return
	}

}

// testSend will create a new link to an address and then a tunnel on top
func testSend(tun tunnel.Tunnel) {
	// dial a new session
	c, err := tun.Dial("test-tunnel")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	log.Log("Dialed connection")

	m := transport.Message{
		Header: map[string]string{
			"test": "header",
		},
	}

	if err := c.Send(&m); err != nil {
		log.Fatal(err)
	}

	log.Log("Sent message")
}

func main() {
	// create a new tunnel client
	tunA := tunnel.NewTunnel(
		tunnel.Address("127.0.0.1:9096"),
		tunnel.Nodes("127.0.0.1:9097"),
	)

	// create a new tunnel server
	tunB := tunnel.NewTunnel(
		tunnel.Address("127.0.0.1:9097"),
	)

	// start tunB
	err := tunB.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer tunB.Close()
	log.Log("Connected tunnel B")

	time.Sleep(time.Millisecond * 50)

	// start tunA
	err = tunA.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer tunA.Close()
	log.Log("Connected tunnel A")

	time.Sleep(time.Millisecond * 50)

	var wg sync.WaitGroup

	// start accepting connections
	// on tunnel A
	wg.Add(1)
	go testAccept(tunA, &wg)
	wg.Wait()

	time.Sleep(time.Millisecond * 50)

	// dial and send via B
	wg.Add(1)
	testSend(tunB)

	// wait until done
	wg.Wait()
}
