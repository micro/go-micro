package network

import (
	"fmt"
	"io"
	"sync"

	"github.com/google/uuid"
	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/network/transport"
	"github.com/micro/go-micro/util/addr"
)

// default network implementation
type network struct {
	options.Options

	// name of the network
	name string

	// transport
	transport transport.Transport
}

type listener struct {
	// start accepting once
	once sync.Once
	// close channel to close the connection
	closed chan bool
	// the listener
	listener transport.Listener
	// the connection queue
	conns chan Conn
}

func (n *network) Create() (*Node, error) {
	ip, err := addr.Extract("")
	if err != nil {
		return nil, err
	}
	return &Node{
		Id:      fmt.Sprintf("%s-%s", n.name, uuid.New().String()),
		Address: ip,
		Metadata: map[string]string{
			"network": n.Name(),
		},
	}, nil
}

func (n *network) Name() string {
	return n.name
}

func (n *network) Connect(node *Node) (Conn, error) {
	c, err := n.transport.Dial(node.Address)
	if err != nil {
		return nil, err
	}
	return newLink(c.(transport.Socket)), nil
}

func (n *network) Listen(node *Node) (Listener, error) {
	l, err := n.transport.Listen(node.Address)
	if err != nil {
		return nil, err
	}
	return newListener(l), nil
}

func (l *listener) process() {
	if err := l.listener.Accept(l.accept); err != nil {
		// close the listener
		l.Close()
	}
}

func (l *listener) accept(sock transport.Socket) {
	// create a new link and pass it through
	link := newLink(sock)

	// send it
	l.conns <- link

	// wait for it to be closed
	select {
	case <-l.closed:
		return
	case <-link.closed:
		return
	}
}

func (l *listener) Address() string {
	return l.listener.Addr()
}

func (l *listener) Close() error {
	select {
	case <-l.closed:
		return nil
	default:
		close(l.closed)
	}
	return nil
}

func (l *listener) Accept() (Conn, error) {
	l.once.Do(func() {
		// TODO: catch the error
		go l.process()
	})
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.closed:
		return nil, io.EOF
	}
}

func newListener(l transport.Listener) *listener {
	return &listener{
		closed:   make(chan bool),
		conns:    make(chan Conn),
		listener: l,
	}
}

func newNetwork(opts ...options.Option) *network {
	options := options.NewOptions(opts...)

	net := &network{
		name:      DefaultName,
		transport: transport.DefaultTransport,
	}

	// get network name
	name, ok := options.Values().Get("network.name")
	if ok {
		net.name = name.(string)
	}

	// get network transport
	t, ok := options.Values().Get("network.transport")
	if ok {
		net.transport = t.(transport.Transport)
	}

	return net
}
