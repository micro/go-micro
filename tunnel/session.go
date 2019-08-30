package tunnel

import (
	"errors"
	"io"

	"github.com/micro/go-micro/transport"
	"github.com/micro/go-micro/util/log"
)

// session is our pseudo session for transport.Socket
type session struct {
	// unique id based on the remote tunnel id
	id string
	// the channel name
	channel string
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
	// outbound marks the session as outbound dialled connection
	outbound bool
	// lookback marks the session as a loopback on the inbound
	loopback bool
	// the link on which this message was received
	link string
	// the error response
	errChan chan error
}

// message is sent over the send channel
type message struct {
	// type of message
	typ string
	// tunnel id
	id string
	// channel name
	channel string
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
	// the error channel
	errChan chan error
}

func (s *session) Remote() string {
	return s.remote
}

func (s *session) Local() string {
	return s.local
}

func (s *session) Id() string {
	return s.session
}

func (s *session) Channel() string {
	return s.channel
}

func (s *session) Send(m *transport.Message) error {
	select {
	case <-s.closed:
		return errors.New("session is closed")
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
		channel:  s.channel,
		session:  s.session,
		outbound: s.outbound,
		loopback: s.loopback,
		data:     data,
		// specify the link on which to send this
		// it will be blank for dialled sessions
		link: s.link,
		// error chan
		errChan: s.errChan,
	}
	log.Debugf("Appending %+v to send backlog", msg)
	s.send <- msg

	// wait for an error response
	select {
	case err := <-msg.errChan:
		return err
	case <-s.closed:
		return io.EOF
	}

	return nil
}

func (s *session) Recv(m *transport.Message) error {
	select {
	case <-s.closed:
		return errors.New("session is closed")
	default:
		// no op
	}
	// recv from backlog
	msg := <-s.recv

	// check the error if one exists
	select {
	case err := <-msg.errChan:
		return err
	default:
	}

	log.Debugf("Received %+v from recv backlog", msg)
	// set message
	*m = *msg.data
	// return nil
	return nil
}

// Close closes the session
func (s *session) Close() error {
	select {
	case <-s.closed:
		// no op
	default:
		close(s.closed)
	}
	return nil
}
