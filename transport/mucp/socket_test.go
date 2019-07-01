package mucp

import (
	"testing"

	"github.com/micro/go-micro/transport"
)

func TestTunnelSocket(t *testing.T) {
	s := &socket{
		id:     "1",
		closed: make(chan bool),
		remote: "remote",
		local:  "local",
		send:   make(chan *message, 1),
		recv:   make(chan *message, 1),
	}

	// check addresses local and remote
	if s.Local() != s.local {
		t.Fatalf("Expected s.Local %s got %s", s.local, s.Local())
	}
	if s.Remote() != s.remote {
		t.Fatalf("Expected s.Remote %s got %s", s.remote, s.Remote())
	}

	// send a message
	s.Send(&transport.Message{Header: map[string]string{}})

	// get sent message
	msg := <-s.send

	if msg.id != s.id {
		t.Fatalf("Expected sent message id %s got %s", s.id, msg.id)
	}

	// recv a message
	msg.data.Header["Foo"] = "bar"
	s.recv <- msg

	m := new(transport.Message)
	s.Recv(m)

	// check header
	if m.Header["Foo"] != "bar" {
		t.Fatalf("Did not receive correct message %+v", m)
	}

	// close the connection
	s.Close()

	// check connection
	err := s.Send(m)
	if err == nil {
		t.Fatal("Expected closed connection")
	}
	err = s.Recv(m)
	if err == nil {
		t.Fatal("Expected closed connection")
	}
}
