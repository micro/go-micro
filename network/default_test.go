package network

import (
	"io"
	"testing"
)

func TestNetwork(t *testing.T) {
	// create a new network
	n := newNetwork()

	// create a new node
	node, err := n.Create()
	if err != nil {
		t.Fatal(err)
	}

	// set ourselves a random port
	node.Address = node.Address + ":0"

	l, err := n.Listen(node)
	if err != nil {
		t.Fatal(err)
	}

	wait := make(chan error)

	go func() {
		var gerr error

		for {
			c, err := l.Accept()
			if err != nil {
				gerr = err
				break
			}
			m := new(Message)
			if err := c.Recv(m); err != nil {
				gerr = err
				break
			}
			if err := c.Send(m); err != nil {
				gerr = err
				break
			}
		}

		wait <- gerr
	}()

	node.Address = l.Address()

	// connect to the node
	conn, err := n.Connect(node)
	if err != nil {
		t.Fatal(err)
	}

	// send a message
	if err := conn.Send(&Message{
		Header: map[string]string{"Foo": "bar"},
		Body:   []byte(`hello world`),
	}); err != nil {
		t.Fatal(err)
	}

	m := new(Message)
	// send a message
	if err := conn.Recv(m); err != nil {
		t.Fatal(err)
	}

	if m.Header["Foo"] != "bar" {
		t.Fatalf("Received unexpected message %+v", m)
	}

	// close the listener
	l.Close()

	// get listener error
	err = <-wait

	if err != io.EOF {
		t.Fatal(err)
	}
}
