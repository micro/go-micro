// Package link provides a measured transport.Socket link
package link

import (
	"io"
	"sync"

	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/transport"
)

type link struct {
	sync.RWMutex

	// the link id
	id string

	// the remote end to dial
	addr string

	// channel used to close the link
	closed chan bool

	// if its connected
	connected bool

	// the transport to use
	transport transport.Transport

	// the send queue to the socket
	sendQueue chan *transport.Message
	// the recv queue to the socket
	recvQueue chan *transport.Message

	// the socket for this link
	socket transport.Socket

	// determines the cost of the link
	// based on queue length and roundtrip
	length int
	weight int
}

func newLink(options options.Options) *link {
	// default values
	var sock transport.Socket
	var addr string
	id := "local"
	tr := transport.DefaultTransport

	lid, ok := options.Values().Get("link.id")
	if ok {
		id = lid.(string)
	}

	laddr, ok := options.Values().Get("link.address")
	if ok {
		addr = laddr.(string)
	}

	ltr, ok := options.Values().Get("link.transport")
	if ok {
		tr = ltr.(transport.Transport)
	}

	lsock, ok := options.Values().Get("link.socket")
	if ok {
		sock = lsock.(transport.Socket)
	}

	l := &link{
		// the remote end to dial
		addr: addr,
		// transport to dial link
		transport: tr,
		// the socket to use
		// this is nil if not specified
		socket: sock,
		// unique id assigned to the link
		id: id,
		// the closed channel used to close the conn
		closed: make(chan bool),
		// then send queue
		sendQueue: make(chan *transport.Message, 128),
		// the receive queue
		recvQueue: make(chan *transport.Message, 128),
	}

	// return the link
	return l
}

// link methods

// process processes messages on the send and receive queues.
func (l *link) process() {
	go func() {
		for {
			m := new(transport.Message)
			if err := l.recv(m); err != nil {
				return
			}

			select {
			case l.recvQueue <- m:
			case <-l.closed:
				return
			}
		}
	}()

	for {
		select {
		case m := <-l.sendQueue:
			if err := l.send(m); err != nil {
				return
			}
		case <-l.closed:
			return
		}
	}
}

// send a message over the link
func (l *link) send(m *transport.Message) error {
	// TODO: measure time taken and calculate length/rate
	// send via the transport socket
	return l.socket.Send(m)
}

// recv a message on the link
func (l *link) recv(m *transport.Message) error {
	if m.Header == nil {
		m.Header = make(map[string]string)
	}
	// receive the transport message
	return l.socket.Recv(m)
}

// Connect attempts to connect to an address and sets the socket
func (l *link) Connect() error {
	l.Lock()
	if l.connected {
		l.Unlock()
		return nil
	}
	defer l.Unlock()

	// replace closed
	l.closed = make(chan bool)

	// assume existing socket
	if len(l.addr) == 0 {
		go l.process()
		return nil
	}

	// dial the endpoint
	c, err := l.transport.Dial(l.addr)
	if err != nil {
		return nil
	}

	// set the socket
	l.socket = c

	// kick start the processing
	go l.process()

	return nil
}

// Close the link
func (l *link) Close() error {
	select {
	case <-l.closed:
		return nil
	default:
		close(l.closed)
		l.Lock()
		l.connected = false
		l.Unlock()
		return l.socket.Close()
	}
}

// returns the node id
func (l *link) Id() string {
	l.RLock()
	defer l.RUnlock()
	return l.id
}

// the remote ip of the link
func (l *link) Remote() string {
	l.RLock()
	defer l.RUnlock()
	return l.socket.Remote()
}

// the local ip of the link
func (l *link) Local() string {
	l.RLock()
	defer l.RUnlock()
	return l.socket.Local()
}

// length/rate of the link
func (l *link) Length() int {
	l.RLock()
	defer l.RUnlock()
	return l.length
}

// weight checks the size of the queues
func (l *link) Weight() int {
	return len(l.sendQueue) + len(l.recvQueue)
}

// Accept accepts a message on the socket
func (l *link) Recv(m *transport.Message) error {
	select {
	case <-l.closed:
		return io.EOF
	case rm := <-l.recvQueue:
		*m = *rm
		return nil
	}
	// never reach
	return nil
}

// Send sends a message on the socket immediately
func (l *link) Send(m *transport.Message) error {
	select {
	case <-l.closed:
		return io.EOF
	case l.sendQueue <- m:
	}
	return nil
}

func (l *link) Status() string {
	select {
	case <-l.closed:
		return "closed"
	default:
		return "connected"
	}
}
