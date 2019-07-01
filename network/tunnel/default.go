package tunnel

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/micro/go-micro/transport"
)

// tun represents a network tunnel
type tun struct {
	// interface to use
	net Interface

	// connect
	mtx       sync.RWMutex
	connected bool

	// the send channel
	send chan *message
	// close channel
	closed chan bool

	// sockets
	sockets map[string]*socket
}

// create new tunnel
func newTunnel(net Interface) *tun {
	return &tun{
		net:     net,
		send:    make(chan *message, 128),
		closed:  make(chan bool),
		sockets: make(map[string]*socket),
	}
}

func (t *tun) getSocket(id string) (*socket, bool) {
	// get the socket
	t.mtx.RLock()
	s, ok := t.sockets[id]
	t.mtx.RUnlock()
	return s, ok
}

func (t *tun) newSocket(id string) *socket {
	// new id if it doesn't exist
	if len(id) == 0 {
		id = uuid.New().String()
	}

	// hash the id
	h := sha256.New()
	h.Write([]byte(id))
	id = fmt.Sprintf("%x", h.Sum(nil))

	// new socket
	s := &socket{
		id:     id,
		closed: make(chan bool),
		recv:   make(chan *message, 128),
		send:   t.send,
	}

	// save socket
	t.mtx.Lock()
	t.sockets[id] = s
	t.mtx.Unlock()

	// return socket
	return s
}

// process outgoing messages
func (t *tun) process() {
	// manage the send buffer
	// all pseudo sockets throw everything down this
	for {
		select {
		case msg := <-t.send:
			nmsg := &Message{
				Header: msg.data.Header,
				Body:   msg.data.Body,
			}

			// set the stream id on the outgoing message
			nmsg.Header["Micro-Stream"] = msg.id

			// send the message via the interface
			if err := t.net.Send(nmsg); err != nil {
				// no op
				// TODO: do something
			}
		case <-t.closed:
			return
		}
	}
}

// process incoming messages
func (t *tun) listen() {
	for {
		// process anything via the net interface
		msg, err := t.net.Recv()
		if err != nil {
			return
		}

		// a stream id
		id := msg.Header["Micro-Stream"]

		// get the socket
		s, exists := t.getSocket(id)
		if !exists {
			// no op
			continue
		}

		// is the socket closed?
		select {
		case <-s.closed:
			// closed
			delete(t.sockets, id)
			continue
		default:
			// process
		}

		// is the socket new?
		select {
		// if its new it will block here
		case <-s.wait:
			// its not new
		default:
			// its new
			// set remote address of the socket
			s.remote = msg.Header["Remote"]
			close(s.wait)
		}

		tmsg := &transport.Message{
			Header: msg.Header,
			Body:   msg.Body,
		}

		// TODO: don't block on queuing
		// append to recv backlog
		s.recv <- &message{id: id, data: tmsg}
	}
}

// Close the tunnel
func (t *tun) Close() error {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if !t.connected {
		return nil
	}

	select {
	case <-t.closed:
		return nil
	default:
		// close all the sockets
		for _, s := range t.sockets {
			s.Close()
		}
		// close the connection
		close(t.closed)
		t.connected = false
	}

	return nil
}

// Connect the tunnel
func (t *tun) Connect() error {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	// already connected
	if t.connected {
		return nil
	}

	// set as connected
	t.connected = true
	// create new close channel
	t.closed = make(chan bool)

	// process messages to be sent
	go t.process()
	// process incoming messages
	go t.listen()

	return nil
}

// Dial an address
func (t *tun) Dial(addr string) (Conn, error) {
	c := t.newSocket(addr)
	// set remote
	c.remote = addr
	// set local
	c.local = t.net.Addr()
	return c, nil
}

func (t *tun) Accept(addr string) (Conn, error) {
	c := t.newSocket(addr)
	// set remote
	c.remote = t.net.Addr()
	// set local
	c.local = addr

	select {
	case <-c.closed:
		return nil, errors.New("error creating socket")
	// wait for the first message
	case <-c.wait:
	}

	// return socket
	return c, nil
}
