package tunnel

import (
	"crypto/cipher"
	"encoding/base32"
	"io"
	"sync"
	"time"

	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/transport"
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
	// the dial timeout
	dialTimeout time.Duration
	// the read timeout
	readTimeout time.Duration
	// the link on which this message was received
	link string
	// the error response
	errChan chan error
	// key for session encryption
	key []byte
	// cipher for session
	gcm cipher.AEAD
	sync.RWMutex
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

func (s *session) sendMsg(msg *message) error {
	select {
	case <-s.closed:
		return io.EOF
	case s.send <- msg:
		return nil
	}
}

func (s *session) wait(msg *message) error {
	// wait for an error response
	select {
	case err := <-msg.errChan:
		if err != nil {
			return err
		}
	case <-s.closed:
		return io.EOF
	}

	return nil
}

// waitFor waits for the message type required until the timeout specified
func (s *session) waitFor(msgType string, timeout time.Duration) (*message, error) {
	now := time.Now()

	after := func(timeout time.Duration) <-chan time.Time {
		if timeout < time.Duration(0) {
			return nil
		}

		// get the delta
		d := time.Since(now)

		// dial timeout minus time since
		wait := timeout - d

		if wait < time.Duration(0) {
			wait = time.Duration(0)
		}

		return time.After(wait)
	}

	// wait for the message type
	for {
		select {
		case msg := <-s.recv:
			// there may be no message type
			if len(msgType) == 0 {
				return msg, nil
			}

			// ignore what we don't want
			if msg.typ != msgType {
				if logger.V(logger.DebugLevel, log) {
					log.Debugf("Tunnel received non %s message in waiting for %s", msg.typ, msgType)
				}
				continue
			}

			// got the message
			return msg, nil
		case <-after(timeout):
			return nil, ErrReadTimeout
		case <-s.closed:
			// check pending message queue
			select {
			case msg := <-s.recv:
				// there may be no message type
				if len(msgType) == 0 {
					return msg, nil
				}

				// ignore what we don't want
				if msg.typ != msgType {
					if logger.V(logger.DebugLevel, log) {
						log.Debugf("Tunnel received non %s message in waiting for %s", msg.typ, msgType)
					}
					continue
				}

				// got the message
				return msg, nil
			default:
				// non blocking
			}
			return nil, io.EOF
		}
	}
}

// Discover attempts to discover the link for a specific channel.
// This is only used by the tunnel.Dial when first connecting.
func (s *session) Discover() error {
	// create a new discovery message for this channel
	msg := s.newMessage("discover")
	// broadcast the message to all links
	msg.mode = Broadcast
	// its an outbound connection since we're dialling
	msg.outbound = true
	// don't set the link since we don't know where it is
	msg.link = ""

	// if multicast then set that as session
	if s.mode == Multicast {
		msg.session = "multicast"
	}

	// send discover message
	if err := s.sendMsg(msg); err != nil {
		return err
	}

	// set time now
	now := time.Now()

	// after strips down the dial timeout
	after := func() time.Duration {
		d := time.Since(now)
		// dial timeout minus time since
		wait := s.dialTimeout - d
		// make sure its always > 0
		if wait < time.Duration(0) {
			return time.Duration(0)
		}
		return wait
	}

	// the discover message is sent out, now
	// wait to hear back about the sent message
	select {
	case <-time.After(after()):
		return ErrDialTimeout
	case err := <-s.errChan:
		if err != nil {
			return err
		}
	}

	// bail early if its not unicast
	// we don't need to wait for the announce
	if s.mode != Unicast {
		s.discovered = true
		s.accepted = true
		return nil
	}

	// wait for announce
	_, err := s.waitFor("announce", after())
	if err != nil {
		return err
	}

	// set discovered
	s.discovered = true

	return nil
}

// Open will fire the open message for the session. This is called by the dialler.
// This is to indicate that we want to create a new session.
func (s *session) Open() error {
	// create a new message
	msg := s.newMessage("open")

	// send open message
	if err := s.sendMsg(msg); err != nil {
		return err
	}

	// wait for an error response for send
	if err := s.wait(msg); err != nil {
		return err
	}

	// now wait for the accept message to be returned
	msg, err := s.waitFor("accept", s.dialTimeout)
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
	if err := s.sendMsg(msg); err != nil {
		return err
	}

	// wait for send response
	return s.wait(msg)
}

// Announce sends an announcement to notify that this session exists.
// This is primarily used by the listener.
func (s *session) Announce() error {
	msg := s.newMessage("announce")
	// we don't need an error back
	msg.errChan = nil
	// announce to all
	msg.mode = Broadcast
	// we don't need the link
	msg.link = ""

	// send announce message
	return s.sendMsg(msg)
}

// Send is used to send a message
func (s *session) Send(m *transport.Message) error {
	var err error

	s.RLock()
	gcm := s.gcm
	s.RUnlock()

	if gcm == nil {
		gcm, err = newCipher(s.key)
		if err != nil {
			return err
		}
		s.Lock()
		s.gcm = gcm
		s.Unlock()
	}
	// encrypt the transport message payload
	body, err := Encrypt(gcm, m.Body)
	if err != nil {
		log.Debugf("failed to encrypt message body: %v", err)
		return err
	}

	// make copy, without rehash and realloc
	data := &transport.Message{
		Header: make(map[string]string, len(m.Header)),
		Body:   body,
	}

	// encrypt all the headers
	for k, v := range m.Header {
		// encrypt the transport message payload
		val, err := Encrypt(s.gcm, []byte(v))
		if err != nil {
			log.Debugf("failed to encrypt message header %s: %v", k, err)
			return err
		}
		// add the encrypted header value
		data.Header[k] = base32.StdEncoding.EncodeToString(val)
	}

	// create a new message
	msg := s.newMessage("session")
	// set the data
	msg.data = data

	// if multicast don't set the link
	if s.mode != Unicast {
		msg.link = ""
	}

	if logger.V(logger.TraceLevel, log) {
		log.Tracef("Appending to send backlog: %v", msg)
	}
	// send the actual message
	if err := s.sendMsg(msg); err != nil {
		return err
	}

	// wait for an error response
	return s.wait(msg)
}

// Recv is used to receive a message
func (s *session) Recv(m *transport.Message) error {
	var msg *message

	msg, err := s.waitFor("", s.readTimeout)
	if err != nil {
		return err
	}

	// check the error if one exists
	select {
	case err := <-msg.errChan:
		return err
	default:
	}

	if logger.V(logger.TraceLevel, log) {
		log.Tracef("Received from recv backlog: %v", msg)
	}

	gcm, err := newCipher([]byte(s.token + s.channel + msg.session))
	if err != nil {
		if logger.V(logger.ErrorLevel, log) {
			log.Errorf("unable to create cipher: %v", err)
		}
		return err
	}

	// decrypt the received payload using the token
	// we have to used msg.session because multicast has a shared
	// session id of "multicast" in this session struct on
	// the listener side
	msg.data.Body, err = Decrypt(gcm, msg.data.Body)
	if err != nil {
		if logger.V(logger.DebugLevel, log) {
			log.Debugf("failed to decrypt message body: %v", err)
		}
		return err
	}

	// dencrypt all the headers
	for k, v := range msg.data.Header {
		// decode the header values
		h, err := base32.StdEncoding.DecodeString(v)
		if err != nil {
			if logger.V(logger.DebugLevel, log) {
				log.Debugf("failed to decode message header %s: %v", k, err)
			}
			return err
		}

		// dencrypt the transport message payload
		val, err := Decrypt(gcm, h)
		if err != nil {
			if logger.V(logger.DebugLevel, log) {
				log.Debugf("failed to decrypt message header %s: %v", k, err)
			}
			return err
		}
		// add decrypted header value
		msg.data.Header[k] = string(val)
	}

	// set the link
	// TODO: decruft, this is only for multicast
	// since the session is now a single session
	// likely provide as part of message.Link()
	msg.data.Header["Micro-Link"] = msg.link

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

		// don't send close on multicast or broadcast
		if s.mode != Unicast {
			return nil
		}

		// append to backlog
		msg := s.newMessage("close")
		// no error response on close
		msg.errChan = nil

		// send the close message
		select {
		case s.send <- msg:
		case <-time.After(time.Millisecond * 10):
		}
	}

	return nil
}
