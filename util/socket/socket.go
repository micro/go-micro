// Package socket provides a pseudo socket
package socket

import (
	"io"

	"github.com/asim/go-micro/v3/transport"
)

// Socket is our pseudo socket for transport.Socket
type Socket struct {
	id string
	// closed
	closed chan bool
	// remote addr
	remote string
	// local addr
	local string
	// send chan
	send chan *transport.Message
	// recv chan
	recv chan *transport.Message
}

func (s *Socket) SetLocal(l string) {
	s.local = l
}

func (s *Socket) SetRemote(r string) {
	s.remote = r
}

// Accept passes a message to the socket which will be processed by the call to Recv
func (s *Socket) Accept(m *transport.Message) error {
	select {
	case s.recv <- m:
		return nil
	case <-s.closed:
		return io.EOF
	}
}

// Process takes the next message off the send queue created by a call to Send
func (s *Socket) Process(m *transport.Message) error {
	select {
	case msg := <-s.send:
		*m = *msg
	case <-s.closed:
		// see if we need to drain
		select {
		case msg := <-s.send:
			*m = *msg
			return nil
		default:
			return io.EOF
		}
	}
	return nil
}

func (s *Socket) Remote() string {
	return s.remote
}

func (s *Socket) Local() string {
	return s.local
}

func (s *Socket) Send(m *transport.Message) error {
	// send a message
	select {
	case s.send <- m:
	case <-s.closed:
		return io.EOF
	}

	return nil
}

func (s *Socket) Recv(m *transport.Message) error {
	// receive a message
	select {
	case msg := <-s.recv:
		// set message
		*m = *msg
	case <-s.closed:
		return io.EOF
	}

	// return nil
	return nil
}

// Close closes the socket
func (s *Socket) Close() error {
	select {
	case <-s.closed:
		// no op
	default:
		close(s.closed)
	}
	return nil
}

// New returns a new pseudo socket which can be used in the place of a transport socket.
// Messages are sent to the socket via Accept and receives from the socket via Process.
// SetLocal/SetRemote should be called before using the socket.
func New(id string) *Socket {
	return &Socket{
		id:     id,
		closed: make(chan bool),
		local:  "local",
		remote: "remote",
		send:   make(chan *transport.Message, 128),
		recv:   make(chan *transport.Message, 128),
	}
}
