package tunnel

import (
	"encoding/hex"
	"errors"
	"io"
	"time"

	"github.com/micro/go-micro/transport"
	"github.com/micro/go-micro/util/log"
)

// session is our pseudo session for transport.Socket
type session struct {
	// the tunnel id
	tunnel string
	// the channel name
	channel string
	// the session id based on Micro.Tunnel-Session
	session string
	// token is the session token
	token string
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
	// if the discovery worked
	discovered bool
	// if the session was accepted
	accepted bool
	// outbound marks the session as outbound dialled connection
	outbound bool
	// lookback marks the session as a loopback on the inbound
	loopback bool
	// mode of the connection
	mode Mode
	// the timeout
	timeout time.Duration
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
	tunnel string
	// channel name
	channel string
	// the session id
	session string
	// outbound marks the message as outbound
	outbound bool
	// loopback marks the message intended for loopback
	loopback bool
	// mode of the connection
	mode Mode
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

func (s *session) Link() string {
	return s.link
}

func (s *session) Id() string {
	return s.session
}

func (s *session) Channel() string {
	return s.channel
}

// newMessage creates a new message based on the session
func (s *session) newMessage(typ string) *message {
	return &message{
		typ:      typ,
		tunnel:   s.tunnel,
		channel:  s.channel,
		session:  s.session,
		outbound: s.outbound,
		loopback: s.loopback,
		mode:     s.mode,
		link:     s.link,
		errChan:  s.errChan,
	}
}

// waitFor waits for the message type required until the timeout specified
func (s *session) waitFor(msgType string, timeout time.Duration) (*message, error) {
	now := time.Now()

	after := func(timeout time.Duration) time.Duration {
		d := time.Since(now)
		// dial timeout minus time since
		wait := timeout - d

		if wait < time.Duration(0) {
			return time.Duration(0)
		}

		return wait
	}

	// wait for the message type
	for {
		select {
		case msg := <-s.recv:
			// ignore what we don't want
			if msg.typ != msgType {
				log.Debugf("Tunnel received non %s message in waiting for %s", msg.typ, msgType)
				continue
			}
			// got the message
			return msg, nil
		case <-time.After(after(timeout)):
			return nil, ErrDialTimeout
		case <-s.closed:
			return nil, io.EOF
		}
	}
}

// Discover attempts to discover the link for a specific channel
func (s *session) Discover() error {
	// create a new discovery message for this channel
	msg := s.newMessage("discover")
	msg.mode = Broadcast
	msg.outbound = true
	msg.link = ""

	// send the discovery message
	s.send <- msg

	// set time now
	now := time.Now()

	after := func() time.Duration {
		d := time.Since(now)
		// dial timeout minus time since
		wait := s.timeout - d
		if wait < time.Duration(0) {
			return time.Duration(0)
		}
		return wait
	}

	// wait to hear back about the sent message
	select {
	case <-time.After(after()):
		return ErrDialTimeout
	case err := <-s.errChan:
		if err != nil {
			return err
		}
	}

	var err error

	// set a new dialTimeout
	dialTimeout := after()

	// set a shorter delay for multicast
	if s.mode != Unicast {
		// shorten this
		dialTimeout = time.Millisecond * 500
	}

	// wait for announce
	_, err = s.waitFor("announce", dialTimeout)

	// if its multicast just go ahead because this is best effort
	if s.mode != Unicast {
		s.discovered = true
		s.accepted = true
		return nil
	}

	if err != nil {
		return err
	}

	// set discovered
	s.discovered = true

	return nil
}

// Open will fire the open message for the session. This is called by the dialler.
func (s *session) Open() error {
	// create a new message
	msg := s.newMessage("open")

	// send open message
	s.send <- msg

	// wait for an error response for send
	select {
	case err := <-msg.errChan:
		if err != nil {
			return err
		}
	case <-s.closed:
		return io.EOF
	}

	// don't wait on multicast/broadcast
	if s.mode == Multicast {
		s.accepted = true
		return nil
	}

	// now wait for the accept
	msg, err := s.waitFor("accept", s.timeout)
	if err != nil {
		return err
	}

	// set to accepted
	s.accepted = true
	// set link
	s.link = msg.link

	return nil
}

// Accept sends the accept response to an open message from a dialled connection
func (s *session) Accept() error {
	msg := s.newMessage("accept")

	// send the accept message
	select {
	case <-s.closed:
		return io.EOF
	case s.send <- msg:
		// no op here
	}

	// don't wait on multicast/broadcast
	if s.mode == Multicast {
		return nil
	}

	// wait for send response
	select {
	case err := <-s.errChan:
		if err != nil {
			return err
		}
	case <-s.closed:
		return io.EOF
	}

	return nil
}

// Announce sends an announcement to notify that this session exists. This is primarily used by the listener.
func (s *session) Announce() error {
	msg := s.newMessage("announce")
	// we don't need an error back
	msg.errChan = nil
	// announce to all
	msg.mode = Broadcast
	// we don't need the link
	msg.link = ""

	select {
	case s.send <- msg:
		return nil
	case <-s.closed:
		return io.EOF
	}
}

// Send is used to send a message
func (s *session) Send(m *transport.Message) error {
	select {
	case <-s.closed:
		return io.EOF
	default:
		// no op
	}

	// encrypt the transport message payload
	body, err := Encrypt(m.Body, s.token+s.channel+s.session)
	if err != nil {
		log.Debugf("failed to encrypt message body: %v", err)
		return err
	}

	// make copy
	data := &transport.Message{
		Header: make(map[string]string),
		Body:   body,
	}

	// encrypt all the headers
	for k, v := range m.Header {
		// encrypt the transport message payload
		val, err := Encrypt([]byte(v), s.token+s.channel+s.session)
		if err != nil {
			log.Debugf("failed to encrypt message header %s: %v", k, err)
			return err
		}
		// hex encode the encrypted header value
		data.Header[k] = hex.EncodeToString(val)
	}

	// create a new message
	msg := s.newMessage("session")
	// set the data
	msg.data = data

	// if multicast don't set the link
	if s.mode == Multicast {
		msg.link = ""
	}

	log.Tracef("Appending %+v to send backlog", msg)
	// send the actual message
	s.send <- msg

	// wait for an error response
	select {
	case err := <-msg.errChan:
		return err
	case <-s.closed:
		return io.EOF
	}
}

// Recv is used to receive a message
func (s *session) Recv(m *transport.Message) error {
	var msg *message

	select {
	case <-s.closed:
		return errors.New("session is closed")
	// recv from backlog
	case msg = <-s.recv:
	}

	// check the error if one exists
	select {
	case err := <-msg.errChan:
		return err
	default:
	}

	//log.Tracef("Received %+v from recv backlog", msg)
	log.Debugf("Received %+v from recv backlog", msg)

	// decrypt the received payload using the token
	body, err := Decrypt(msg.data.Body, s.token+s.channel+s.session)
	if err != nil {
		log.Debugf("failed to decrypt message body: %v", err)
		return err
	}
	msg.data.Body = body

	// encrypt all the headers
	for k, v := range msg.data.Header {
		// hex decode the header values
		h, err := hex.DecodeString(v)
		if err != nil {
			log.Debugf("failed to decode message header %s: %v", k, err)
			return err
		}
		// encrypt the transport message payload
		val, err := Decrypt([]byte(h), s.token+s.channel+s.session)
		if err != nil {
			log.Debugf("failed to decrypt message header %s: %v", k, err)
			return err
		}
		// hex encode the encrypted header value
		msg.data.Header[k] = string(val)
	}

	// set message
	*m = *msg.data
	// return nil
	return nil
}

// Close closes the session by sending a close message
func (s *session) Close() error {
	select {
	case <-s.closed:
		// no op
	default:
		close(s.closed)

		// append to backlog
		msg := s.newMessage("close")
		// no error response on close
		msg.errChan = nil

		// send the close message
		select {
		case s.send <- msg:
		default:
		}
	}

	return nil
}
