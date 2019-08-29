package tunnel

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/transport"
	"github.com/micro/go-micro/util/log"
)

var (
	// KeepAliveTime defines time interval we send keepalive messages to outbound links
	KeepAliveTime = 30 * time.Second
	// ReconnectTime defines time interval we periodically attempt to reconnect dead links
	ReconnectTime = 5 * time.Second
)

// tun represents a network tunnel
type tun struct {
	options Options

	sync.RWMutex

	// tunnel token
	token string

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
	// unique id of this link e.g uuid
	// which we define for ourselves
	id string
	// whether its a loopback connection
	// this flag is used by the transport listener
	// which accepts inbound quic connections
	loopback bool
	// whether its actually connected
	// dialled side sets it to connected
	// after sending the message. the
	// listener waits for the connect
	connected bool
	// the last time we received a keepalive
	// on this link from the remote side
	lastKeepAlive time.Time
}

// create new tunnel on top of a link
func newTunnel(opts ...Option) *tun {
	options := DefaultOptions()
	for _, o := range opts {
		o(&options)
	}

	return &tun{
		options: options,
		token:   uuid.New().String(),
		send:    make(chan *message, 128),
		closed:  make(chan bool),
		sockets: make(map[string]*socket),
		links:   make(map[string]*link),
	}
}

// Init initializes tunnel options
func (t *tun) Init(opts ...Option) error {
	t.Lock()
	defer t.Unlock()
	for _, o := range opts {
		o(&t.options)
	}
	return nil
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

// monitor monitors outbound links and attempts to reconnect to the failed ones
func (t *tun) monitor() {
	reconnect := time.NewTicker(ReconnectTime)
	defer reconnect.Stop()

	for {
		select {
		case <-t.closed:
			return
		case <-reconnect.C:
			for _, node := range t.options.Nodes {
				t.Lock()
				if _, ok := t.links[node]; !ok {
					link, err := t.setupLink(node)
					if err != nil {
						log.Debugf("Tunnel failed to setup node link to %s: %v", node, err)
						t.Unlock()
						continue
					}
					t.links[node] = link
				}
				t.Unlock()
			}
		}
	}
}

// process outgoing messages sent by all local sockets
func (t *tun) process() {
	// manage the send buffer
	// all pseudo sockets throw everything down this
	for {
		select {
		case msg := <-t.send:
			newMsg := &transport.Message{
				Header: make(map[string]string),
				Body:   msg.data.Body,
			}

			for k, v := range msg.data.Header {
				newMsg.Header[k] = v
			}

			// set message head
			newMsg.Header["Micro-Tunnel"] = msg.typ

			// set the tunnel id on the outgoing message
			newMsg.Header["Micro-Tunnel-Id"] = msg.id

			// set the session id
			newMsg.Header["Micro-Tunnel-Session"] = msg.session

			// set the tunnel token
			newMsg.Header["Micro-Tunnel-Token"] = t.token

			// send the message via the interface
			t.Lock()

			if len(t.links) == 0 {
				log.Debugf("No links to send to")
			}

			for node, link := range t.links {
				// if the link is not connected skip it
				if !link.connected {
					log.Debugf("Link for node %s not connected", node)
					continue
				}

				// if we're picking the link check the id
				// this is where we explicitly set the link
				// in a message received via the listen method
				if len(msg.link) > 0 && link.id != msg.link {
					continue
				}

				// if the link was a loopback accepted connection
				// and the message is being sent outbound via
				// a dialled connection don't use this link
				if link.loopback && msg.outbound {
					continue
				}

				// if the message was being returned by the loopback listener
				// send it back up the loopback link only
				if msg.loopback && !link.loopback {
					continue
				}

				// send the message via the current link
				log.Debugf("Sending %+v to %s", newMsg, node)
				if err := link.Send(newMsg); err != nil {
					log.Debugf("Tunnel error sending %+v to %s: %v", newMsg, node, err)
					delete(t.links, node)
					continue
				}
			}

			t.Unlock()
		case <-t.closed:
			return
		}
	}
}

// process incoming messages
func (t *tun) listen(link *link) {
	// remove the link on exit
	defer func() {
		log.Debugf("Tunnel deleting connection from %s", link.Remote())
		t.Lock()
		delete(t.links, link.Remote())
		t.Unlock()
	}()

	// let us know if its a loopback
	var loopback bool

	for {
		// process anything via the net interface
		msg := new(transport.Message)
		if err := link.Recv(msg); err != nil {
			log.Debugf("Tunnel link %s receive error: %#v", link.Remote(), err)
			return
		}

		switch msg.Header["Micro-Tunnel"] {
		case "connect":
			log.Debugf("Tunnel link %s received connect message", link.Remote())
			// check the Micro-Tunnel-Token
			token, ok := msg.Header["Micro-Tunnel-Token"]
			if !ok {
				continue
			}

			// are we connecting to ourselves?
			if token == t.token {
				link.loopback = true
				loopback = true
			}

			// set as connected
			link.connected = true

			// save the link once connected
			t.Lock()
			t.links[link.Remote()] = link
			t.Unlock()

			// nothing more to do
			continue
		case "close":
			log.Debugf("Tunnel link %s closing connection", link.Remote())
			// TODO: handle the close message
			// maybe report io.EOF or kill the link
			return
		case "keepalive":
			log.Debugf("Tunnel link %s received keepalive", link.Remote())
			t.Lock()
			// save the keepalive
			link.lastKeepAlive = time.Now()
			t.Unlock()
			continue
		case "message":
			// process message
			log.Debugf("Received %+v from %s", msg, link.Remote())
		default:
			// blackhole it
			continue
		}

		// if its not connected throw away the link
		if !link.connected {
			log.Debugf("Tunnel link %s not connected", link.id)
			return
		}

		// strip message header
		delete(msg.Header, "Micro-Tunnel")

		// the tunnel id
		id := msg.Header["Micro-Tunnel-Id"]
		delete(msg.Header, "Micro-Tunnel-Id")

		// the session id
		session := msg.Header["Micro-Tunnel-Session"]
		delete(msg.Header, "Micro-Tunnel-Session")

		// strip token header
		delete(msg.Header, "Micro-Tunnel-Token")

		// if the session id is blank there's nothing we can do
		// TODO: check this is the case, is there any reason
		// why we'd have a blank session? Is the tunnel
		// used for some other purpose?
		if len(id) == 0 || len(session) == 0 {
			continue
		}

		var s *socket
		var exists bool

		// If its a loopback connection then we've enabled link direction
		// listening side is used for listening, the dialling side for dialling
		switch {
		case loopback:
			s, exists = t.getSocket(id, "listener")
		default:
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

		// bail if no socket has been found
		if !exists {
			log.Debugf("Tunnel skipping no socket exists")
			// drop it, we don't care about
			// messages we don't know about
			continue
		}
		log.Debugf("Tunnel using socket %s %s", s.id, s.session)

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
			id:       id,
			session:  session,
			data:     tmsg,
			link:     link.id,
			loopback: loopback,
		}

		// append to recv backlog
		// we don't block if we can't pass it on
		select {
		case s.recv <- imsg:
		default:
		}
	}
}

// keepalive periodically sends keepalive messages to link
func (t *tun) keepalive(link *link) {
	keepalive := time.NewTicker(KeepAliveTime)
	defer keepalive.Stop()

	for {
		select {
		case <-t.closed:
			return
		case <-keepalive.C:
			// send keepalive message
			log.Debugf("Tunnel sending keepalive to link: %v", link.Remote())
			if err := link.Send(&transport.Message{
				Header: map[string]string{
					"Micro-Tunnel":       "keepalive",
					"Micro-Tunnel-Token": t.token,
				},
			}); err != nil {
				log.Debugf("Error sending keepalive to link %v: %v", link.Remote(), err)
				t.Lock()
				delete(t.links, link.Remote())
				t.Unlock()
				return
			}
		}
	}
}

// setupLink connects to node and returns link if successful
// It returns error if the link failed to be established
func (t *tun) setupLink(node string) (*link, error) {
	log.Debugf("Tunnel setting up link: %s", node)
	c, err := t.options.Transport.Dial(node)
	if err != nil {
		log.Debugf("Tunnel failed to connect to %s: %v", node, err)
		return nil, err
	}
	log.Debugf("Tunnel connected to %s", node)

	if err := c.Send(&transport.Message{
		Header: map[string]string{
			"Micro-Tunnel":       "connect",
			"Micro-Tunnel-Token": t.token,
		},
	}); err != nil {
		return nil, err
	}

	// create a new link
	link := newLink(c)
	link.connected = true
	// we made the outbound connection
	// and sent the connect message

	// process incoming messages
	go t.listen(link)

	// start keepalive monitor
	go t.keepalive(link)

	return link, nil
}

// connect the tunnel to all the nodes and listen for incoming tunnel connections
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
			log.Debugf("Tunnel accepted connection from %s", sock.Remote())

			// create a new link
			link := newLink(sock)

			// listen for inbound messages.
			// only save the link once connected.
			// we do this inside liste
			t.listen(link)
		})

		t.RLock()
		defer t.RUnlock()

		// still connected but the tunnel died
		if err != nil && t.connected {
			log.Logf("Tunnel listener died: %v", err)
		}
	}()

	for _, node := range t.options.Nodes {
		// skip zero length nodes
		if len(node) == 0 {
			continue
		}

		// connect to node and return link
		link, err := t.setupLink(node)
		if err != nil {
			log.Debugf("Tunnel failed to establish node link to %s: %v", node, err)
			continue
		}

		// save the link
		t.links[node] = link
	}

	// process outbound messages to be sent
	// process sends to all links
	go t.process()

	// monitor links
	go t.monitor()

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

func (t *tun) close() error {
	// close all the links
	for node, link := range t.links {
		link.Send(&transport.Message{
			Header: map[string]string{
				"Micro-Tunnel":       "close",
				"Micro-Tunnel-Token": t.token,
			},
		})
		link.Close()
		delete(t.links, node)
	}

	// close the listener
	return t.listener.Close()
}

func (t *tun) Address() string {
	t.RLock()
	defer t.RUnlock()

	if !t.connected {
		return t.options.Address
	}

	return t.listener.Addr()
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
		for id, s := range t.sockets {
			s.Close()
			delete(t.sockets, id)
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

// Dial an address
func (t *tun) Dial(addr string) (Conn, error) {
	log.Debugf("Tunnel dialing %s", addr)
	c, ok := t.newSocket(addr, t.newSession())
	if !ok {
		return nil, errors.New("error dialing " + addr)
	}
	// set remote
	c.remote = addr
	// set local
	c.local = "local"
	// outbound socket
	c.outbound = true

	return c, nil
}

// Accept a connection on the address
func (t *tun) Listen(addr string) (Listener, error) {
	log.Debugf("Tunnel listening on %s", addr)
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

func (t *tun) String() string {
	return "mucp"
}
