package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/micro/go-micro/v2/transport"
	"github.com/micro/go-micro/v2/tunnel"
)

var (
	address = flag.String("address", ":10001", "tunnel address")
	nodes   = flag.String("nodes", "", "tunnel nodes")
	channel = flag.String("channel", "default", "the channel")
)

func readLoop(c tunnel.Session, readChan chan *transport.Message) {
	for {
		m := new(transport.Message)
		if err := c.Recv(m); err != nil {
			return
		}

		// fire them into the channel
		select {
		case readChan <- m:
		default:
		}
	}
}

func writeLoop(c tunnel.Session, sendChan chan *transport.Message) {
	for {
		m := <-sendChan

		// don't relay back to sender
		if c.Id() == m.Header["Session"] {
			continue
		}

		// send messages
		if err := c.Send(m); err != nil {
			return
		}
	}
}

func serverAccept(l tunnel.Listener, sendChan chan *transport.Message) {
	connCh := make(chan chan *transport.Message)

	go func() {
		conns := make(map[string]chan *transport.Message)

		for {
			// send all things we're writing to all connections
			select {
			case b := <-sendChan:
				for _, c := range conns {
					select {
					case c <- b:
					default:
					}
				}
			// append new connections
			case c := <-connCh:
				conns[uuid.New().String()] = c
			}
		}
	}()

	for {
		c, err := l.Accept()
		if err != nil {
			return
		}

		fmt.Println("Accepting new connection")

		// pass to reader
		rch := make(chan *transport.Message)
		connCh <- rch

		// send anything we receive
		go readLoop(c, sendChan)

		// write our messages to conn
		go writeLoop(c, rch)
	}
}

func main() {
	flag.Parse()

	// create a tunnel
	tun := tunnel.NewTunnel(
		tunnel.Address(*address),
		tunnel.Nodes(*nodes),
	)
	if err := tun.Connect(); err != nil {
		fmt.Println(err)
		return
	}
	defer tun.Close()

	// listen for inbound messages on the channel
	l, err := tun.Listen(*channel)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()

	sendChan := make(chan *transport.Message)
	printChan := make(chan *transport.Message)

	// accept the messages on the channel
	go serverAccept(l, sendChan)

	// client code

	// dial an outbound connection for the channel
	c, err := tun.Dial(*channel)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer c.Close()

	// write the things we get back
	go func() {
		for m := range printChan {
			if m.Header["Session"] == c.Id() {
				continue
			}
			fmt.Println(string(m.Body))
		}
	}()

	// read and print what we get back on the dialled conn
	go readLoop(c, printChan)

	// read input and send over the tunnel
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		m := &transport.Message{
			Header: map[string]string{
				"Session": c.Id(),
			},
			Body: scanner.Bytes(),
		}

		// send to all listeners
		sendChan <- m

		// send it also
		c.Send(m)
	}
}
