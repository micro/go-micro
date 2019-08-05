package tunnel

import (
	"errors"

	"github.com/alexapps/go-micro/transport"
)

// socket is our pseudo socket for transport.Socket
type socket struct {
	// socket id based on Micro-Tunnel
	id string
	// the session id based on Micro.Tunnel-Session
	session string
	// closed
	closed chan bool
	// remote addr
	remote string
	// local addr
	local string
	// send chan
	send chan *message
	// recv chan
	recv chan *message
	// wait until we have a connection
	wait chan bool
}

// message is sent over the send channel
type message struct {
	// tunnel id
	id string
	// the session id
	session string
	// transport data
	data *transport.Message
}

func (s *socket) Remote() string {
	return s.remote
}

func (s *socket) Local() string {
	return s.local
}

func (s *socket) Id() string {
	return s.id
}

func (s *socket) Session() string {
	return s.session
}

func (s *socket) Send(m *transport.Message) error {
	select {
	case <-s.closed:
		return errors.New("socket is closed")
	default:
		// no op
	}
	// append to backlog
	s.send <- &message{id: s.id, session: s.session, data: m}
	return nil
}

func (s *socket) Recv(m *transport.Message) error {
	select {
	case <-s.closed:
		return errors.New("socket is closed")
	default:
		// no op
	}
	// recv from backlog
	msg := <-s.recv
	// set message
	*m = *msg.data
	// return nil
	return nil
}

func (s *socket) Close() error {
	select {
	case <-s.closed:
		// no op
	default:
		close(s.closed)
	}
	return nil
}
