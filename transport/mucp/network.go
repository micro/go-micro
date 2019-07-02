// Package mucp provides a mucp network transport
package mucp

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"

	"github.com/micro/go-micro/network"
	"github.com/micro/go-micro/transport"
)

type networkKey struct{}

// Transport is a mucp transport. It should only
// be created with NewTransport and cast to
// *Transport if there's a need to close it.
type Transport struct {
	options transport.Options

	// the network interface
	network network.Network

	// protect all the things
	sync.RWMutex

	// connect
	connected bool
	// connected node
	node network.Node
	// the send channel
	send chan *message
	// close channel
	closed chan bool

	// sockets
	sockets map[string]*socket
	// listeners
	listeners map[string]*listener
}

func (n *Transport) newListener(addr string) *listener {
	// hash the id
	h := sha256.New()
	h.Write([]byte(addr))
	id := fmt.Sprintf("%x", h.Sum(nil))

	// create the listener
	l := &listener{
		id:     id,
		addr:   addr,
		closed: make(chan bool),
		accept: make(chan *socket, 128),
	}

	// save it
	n.Lock()
	n.listeners[id] = l
	n.Unlock()

	return l
}

func (n *Transport) getListener(id string) (*listener, bool) {
	// get the listener
	n.RLock()
	s, ok := n.listeners[id]
	n.RUnlock()
	return s, ok
}

func (n *Transport) getSocket(id string) (*socket, bool) {
	// get the socket
	n.RLock()
	s, ok := n.sockets[id]
	n.RUnlock()
	return s, ok
}

func (n *Transport) newSocket(id string) *socket {
	// hash the id
	h := sha256.New()
	h.Write([]byte(id))
	id = fmt.Sprintf("%x", h.Sum(nil))

	// new socket
	s := &socket{
		id:     id,
		closed: make(chan bool),
		recv:   make(chan *message, 128),
		send:   n.send,
	}

	// save socket
	n.Lock()
	n.sockets[id] = s
	n.Unlock()

	// return socket
	return s
}

// process outgoing messages
func (n *Transport) process() {
	// manage the send buffer
	// all pseudo sockets throw everything down this
	for {
		select {
		case msg := <-n.send:
			netmsg := &network.Message{
				Header: msg.data.Header,
				Body:   msg.data.Body,
			}

			// set the stream id on the outgoing message
			netmsg.Header["Micro-Stream"] = msg.id

			// send the message via the interface
			if err := n.node.Send(netmsg); err != nil {
				// no op
				// TODO: do something
			}
		case <-n.closed:
			return
		}
	}
}

// process incoming messages
func (n *Transport) listen() {
	for {
		// process anything via the net interface
		msg, err := n.node.Accept()
		if err != nil {
			return
		}

		// a stream id
		id := msg.Header["Micro-Stream"]

		// get the socket
		s, exists := n.getSocket(id)
		if !exists {
			// get the listener
			l, ok := n.getListener(id)
			// there's no socket and there's no listener
			if !ok {
				continue
			}

			// listener is closed
			select {
			case <-l.closed:
				// delete it
				n.Lock()
				delete(n.listeners, l.id)
				n.Unlock()
				continue
			default:
			}

			// no socket, create one
			s = n.newSocket(id)
			// set remote address
			s.remote = msg.Header["Remote"]

			// drop that to the listener
			// TODO: non blocking
			l.accept <- s
		}

		// is the socket closed?
		select {
		case <-s.closed:
			// closed
			delete(n.sockets, id)
			continue
		default:
			// process
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

func (n *Transport) Init(opts ...transport.Option) error {
	for _, o := range opts {
		o(&n.options)
	}
	return nil
}

func (n *Transport) Options() transport.Options {
	return n.options
}

// Close the tunnel
func (n *Transport) Close() error {
	n.Lock()
	defer n.Unlock()

	if !n.connected {
		return nil
	}

	select {
	case <-n.closed:
		return nil
	default:
		// close all the sockets
		for _, s := range n.sockets {
			s.Close()
		}
		for _, l := range n.listeners {
			l.Close()
		}
		// close the connection
		close(n.closed)
		// close node connection
		n.node.Close()
		// reset connected
		n.connected = false
	}

	return nil
}

// Connect the tunnel
func (n *Transport) Connect() error {
	n.Lock()
	defer n.Unlock()

	// already connected
	if n.connected {
		return nil
	}

	// get a new node
	node, err := n.network.Connect()
	if err != nil {
		return err
	}

	// set as connected
	n.connected = true
	// create new close channel
	n.closed = make(chan bool)
	// save node
	n.node = node

	// process messages to be sent
	go n.process()
	// process incoming messages
	go n.listen()

	return nil
}

// Dial an address
func (n *Transport) Dial(addr string, opts ...transport.DialOption) (transport.Client, error) {
	if err := n.Connect(); err != nil {
		return nil, err
	}

	// create new socket
	s := n.newSocket(addr)
	// set remote
	s.remote = addr
	// set local
	n.RLock()
	s.local = n.node.Address()
	n.RUnlock()

	return s, nil
}

func (n *Transport) Listen(addr string, opts ...transport.ListenOption) (transport.Listener, error) {
	// check existing listeners
	n.RLock()
	for _, l := range n.listeners {
		if l.addr == addr {
			n.RUnlock()
			return nil, errors.New("already listening on " + addr)
		}
	}
	n.RUnlock()

	// try to connect to the network
	if err := n.Connect(); err != nil {
		return nil, err
	}

	return n.newListener(addr), nil
}

func (n *Transport) String() string {
	return "network"
}

// NewTransport creates a new network transport
func NewTransport(opts ...transport.Option) transport.Transport {
	options := transport.Options{
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&options)
	}

	// get the network interface
	n, ok := options.Context.Value(networkKey{}).(network.Network)
	if !ok {
		n = network.DefaultNetwork
	}

	return &Transport{
		options: options,
		network: n,
		send:    make(chan *message, 128),
		closed:  make(chan bool),
		sockets: make(map[string]*socket),
	}
}

// WithNetwork sets the network interface
func WithNetwork(n network.Network) transport.Option {
	return func(o *transport.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, networkKey{}, n)
	}
}
