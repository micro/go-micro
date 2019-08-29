package tunnel

import (
	"errors"

	"github.com/micro/go-micro/transport"
	"github.com/micro/go-micro/util/log"
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
	// outbound marks the socket as outbound dialled connection
	outbound bool
	// lookback marks the socket as a loopback on the inbound
	loopback bool
	// the link on which this message was received
	link string
}

// message is sent over the send channel
type message struct {
	// type of message
	typ string
	// tunnel id
	id string
	// the session id
	session string
	// outbound marks the message as outbound
	outbound bool
	// loopback marks the message intended for loopback
	loopback bool
	// the link to send the message on
	link string
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

	// make copy
	data := &transport.Message{
		Header: make(map[string]string),
		Body:   m.Body,
	}

	for k, v := range m.Header {
		data.Header[k] = v
	}

	// append to backlog
	msg := &message{
		typ:      "message",
		id:       s.id,
		session:  s.session,
		outbound: s.outbound,
		loopback: s.loopback,
		data:     data,
		// specify the link on which to send this
		// it will be blank for dialled sockets
		link: s.link,
	}
	log.Debugf("Appending %+v to send backlog", msg)
	s.send <- msg
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
	log.Debugf("Received %+v from recv backlog", msg)
	// set message
	*m = *msg.data
	// return nil
	return nil
}

// Close closes the socket
func (s *socket) Close() error {
	select {
	case <-s.closed:
		// no op
	default:
		close(s.closed)
	}
	return nil
}
