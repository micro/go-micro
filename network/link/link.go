package link

import (
	"errors"
	"io"
	"sync"

	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/transport"
)

// Link is a layer ontop of a transport socket with the
// buffering send and recv queue's with the ability to
// measure the actual transport link and reconnect.
type Link interface {
	// provides the transport.Socket interface
	transport.Socket
	// Connect connects the link. It must be called first.
	Connect() error
	// Id of the link likely a uuid if not specified
	Id() string
	// Status of the link
	Status() string
	// Depth of the buffers
	Weight() int
	// Rate of the link
	Length() int
}

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

var (
	ErrLinkClosed = errors.New("link closed")
)

func newLink(options options.Options) *link {
	// default values
	id := "local"
	addr := "127.0.0.1:10001"
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

	l := &link{
		// the remote end to dial
		addr: addr,
		// transport to dial link
		transport: tr,
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

// process processes messages on the send queue.
// these are messages to be sent to the remote side.
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

func (l *link) Connect() error {
	l.Lock()
	if l.connected {
		l.Unlock()
		return nil
	}

	// replace closed
	l.closed = make(chan bool)

	// dial the endpoint
	c, err := l.transport.Dial(l.addr)
	if err != nil {
		l.Unlock()
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

func (l *link) Remote() string {
	l.RLock()
	defer l.RUnlock()
	return l.socket.Remote()
}

func (l *link) Local() string {
	l.RLock()
	defer l.RUnlock()
	return l.socket.Local()
}

func (l *link) Length() int {
	l.RLock()
	defer l.RUnlock()
	return l.length
}

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

// NewLink creates a new link on top of a socket
func NewLink(opts ...options.Option) Link {
	return newLink(options.NewOptions(opts...))
}

// Sets the link id
func Id(id string) options.Option {
	return options.WithValue("link.id", id)
}

// The address to use for the link
func Address(a string) options.Option {
	return options.WithValue("link.address", a)
}

// The transport to use for the link
func Transport(t transport.Transport) options.Option {
	return options.WithValue("link.transport", t)
}
