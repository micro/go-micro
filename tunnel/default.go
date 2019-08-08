package tunnel

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/micro/go-micro/transport"
	"github.com/micro/go-micro/util/log"
)

// tun represents a network tunnel
type tun struct {
	options Options

	sync.RWMutex

	// to indicate if we're connected or not
	connected bool

	// the send channel for all messages
	send chan *message

	// close channel
	closed chan bool

	// a map of sockets based on Micro-Tunnel-Id
	sockets map[string]*socket

	// outbound links
	links map[string]*link

	// listener
	listener transport.Listener
}

type link struct {
	transport.Socket
	id string
}

// create new tunnel on top of a link
func newTunnel(opts ...Option) *tun {
	options := DefaultOptions()
	for _, o := range opts {
		o(&options)
	}

	return &tun{
		options: options,
		send:    make(chan *message, 128),
		closed:  make(chan bool),
		sockets: make(map[string]*socket),
		links:   make(map[string]*link),
	}
}

// getSocket returns a socket from the internal socket map.
// It does this based on the Micro-Tunnel-Id and Micro-Tunnel-Session
func (t *tun) getSocket(id, session string) (*socket, bool) {
	// get the socket
	t.RLock()
	s, ok := t.sockets[id+session]
	t.RUnlock()
	return s, ok
}

// newSocket creates a new socket and saves it
func (t *tun) newSocket(id, session string) (*socket, bool) {
	// hash the id
	h := sha256.New()
	h.Write([]byte(id))
	id = fmt.Sprintf("%x", h.Sum(nil))

	// new socket
	s := &socket{
		id:      id,
		session: session,
		closed:  make(chan bool),
		recv:    make(chan *message, 128),
		send:    t.send,
		wait:    make(chan bool),
	}

	// save socket
	t.Lock()
	_, ok := t.sockets[id+session]
	if ok {
		// socket already exists
		t.Unlock()
		return nil, false
	}
	t.sockets[id+session] = s
	t.Unlock()

	// return socket
	return s, true
}

// TODO: use tunnel id as part of the session
func (t *tun) newSession() string {
	return uuid.New().String()
}

// process outgoing messages sent by all local sockets
func (t *tun) process() {
	// manage the send buffer
	// all pseudo sockets throw everything down this
	for {
		select {
		case msg := <-t.send:
			nmsg := &transport.Message{
				Header: msg.data.Header,
				Body:   msg.data.Body,
			}

			// set the tunnel id on the outgoing message
			nmsg.Header["Micro-Tunnel-Id"] = msg.id

			// set the session id
			nmsg.Header["Micro-Tunnel-Session"] = msg.session

			// send the message via the interface
			t.RLock()
			for _, link := range t.links {
				link.Send(nmsg)
			}
			t.RUnlock()
		case <-t.closed:
			return
		}
	}
}

// process incoming messages
func (t *tun) listen(link transport.Socket, listener bool) {
	for {
		// process anything via the net interface
		msg := new(transport.Message)
		err := link.Recv(msg)
		if err != nil {
			return
		}

		switch msg.Header["Micro-Tunnel"] {
		case "connect", "close":
			continue
		}

		// the tunnel id
		id := msg.Header["Micro-Tunnel-Id"]

		// the session id
		session := msg.Header["Micro-Tunnel-Session"]

		// if the session id is blank there's nothing we can do
		// TODO: check this is the case, is there any reason
		// why we'd have a blank session? Is the tunnel
		// used for some other purpose?
		if len(id) == 0 || len(session) == 0 {
			continue
		}

		var s *socket
		var exists bool

		// if its a local listener then we use that as the session id
		// e.g we're using a loopback connecting to ourselves
		if listener {
			s, exists = t.getSocket(id, "listener")
		} else {
			// get the socket based on the tunnel id and session
			// this could be something we dialed in which case
			// we have a session for it otherwise its a listener
			s, exists = t.getSocket(id, session)
			if !exists {
				// try get it based on just the tunnel id
				// the assumption here is that a listener
				// has no session but its set a listener session
				s, exists = t.getSocket(id, "listener")
			}
		}

		// no socket in existence
		if !exists {
			// drop it, we don't care about
			// messages we don't know about
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
		// if its new the socket is actually blocked waiting
		// for a connection. so we check if its waiting.
		case <-s.wait:
		// if its waiting e.g its new then we close it
		default:
			// set remote address of the socket
			s.remote = msg.Header["Remote"]
			close(s.wait)
		}

		// construct a new transport message
		tmsg := &transport.Message{
			Header: msg.Header,
			Body:   msg.Body,
		}

		// construct the internal message
		imsg := &message{
			id:      id,
			session: session,
			data:    tmsg,
		}

		// append to recv backlog
		// we don't block if we can't pass it on
		select {
		case s.recv <- imsg:
		default:
		}
	}
}

func (t *tun) connect() error {
	l, err := t.options.Transport.Listen(t.options.Address)
	if err != nil {
		return err
	}

	// save the listener
	t.listener = l

	go func() {
		// accept inbound connections
		err := l.Accept(func(sock transport.Socket) {
			// save the link
			id := uuid.New().String()
			t.Lock()
			t.links[id] = &link{
				Socket: sock,
				id:     id,
			}
			t.Unlock()

			// delete the link
			defer func() {
				t.Lock()
				delete(t.links, id)
				t.Unlock()
			}()

			// listen for inbound messages
			t.listen(sock, true)
		})

		t.Lock()
		defer t.Unlock()

		// still connected but the tunnel died
		if err != nil && t.connected {
			log.Logf("Tunnel listener died: %v", err)
		}
	}()

	for _, node := range t.options.Nodes {
		c, err := t.options.Transport.Dial(node)
		if err != nil {
			log.Debugf("Tunnel failed to connect to %s: %v", node, err)
			continue
		}

		if err := c.Send(&transport.Message{
			Header: map[string]string{
				"Micro-Tunnel": "connect",
			},
		}); err != nil {
			continue
		}

		// process incoming messages
		go t.listen(c, false)

		// save the link
		id := uuid.New().String()
		t.links[id] = &link{
			Socket: c,
			id:     id,
		}
	}

	// process outbound messages to be sent
	// process sends to all links
	go t.process()

	return nil
}

func (t *tun) close() error {
	// close all the links
	for id, link := range t.links {
		link.Send(&transport.Message{
			Header: map[string]string{
				"Micro-Tunnel": "close",
			},
		})
		link.Close()
		delete(t.links, id)
	}

	// close the listener
	return t.listener.Close()
}

// Close the tunnel
func (t *tun) Close() error {
	t.Lock()
	defer t.Unlock()

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

		// send a close message
		// we don't close the link
		// just the tunnel
		return t.close()
	}

	return nil
}

// Connect the tunnel
func (t *tun) Connect() error {
	t.Lock()
	defer t.Unlock()

	// already connected
	if t.connected {
		return nil
	}

	// send the connect message
	if err := t.connect(); err != nil {
		return err
	}

	// set as connected
	t.connected = true
	// create new close channel
	t.closed = make(chan bool)

	return nil
}

func (t *tun) Init(opts ...Option) error {
	for _, o := range opts {
		o(&t.options)
	}
	return nil
}

// Dial an address
func (t *tun) Dial(addr string) (Conn, error) {
	c, ok := t.newSocket(addr, t.newSession())
	if !ok {
		return nil, errors.New("error dialing " + addr)
	}

	// set remote
	c.remote = addr
	// set local
	c.local = "local"

	return c, nil
}

// Accept a connection on the address
func (t *tun) Listen(addr string) (Listener, error) {
	// create a new socket by hashing the address
	c, ok := t.newSocket(addr, "listener")
	if !ok {
		return nil, errors.New("already listening on " + addr)
	}

	// set remote. it will be replaced by the first message received
	c.remote = "remote"
	// set local
	c.local = addr

	tl := &tunListener{
		addr: addr,
		// the accept channel
		accept: make(chan *socket, 128),
		// the channel to close
		closed: make(chan bool),
		// tunnel closed channel
		tunClosed: t.closed,
		// the connection
		conn: c,
		// the listener socket
		socket: c,
	}

	// this kicks off the internal message processor
	// for the listener so it can create pseudo sockets
	// per session if they do not exist or pass messages
	// to the existign sessions
	go tl.process()

	// return the listener
	return tl, nil
}
