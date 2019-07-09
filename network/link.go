package network

import (
	"errors"
	"io"
	"sync"

	"github.com/google/uuid"
	"github.com/micro/go-micro/transport"
)

type link struct {
	closed chan bool

	sync.RWMutex

	// the link id
	id string

	// the send queue to the socket
	sendQueue chan *Message
	// the recv queue to the socket
	recvQueue chan *Message

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

func newLink(sock transport.Socket) *link {
	l := &link{
		id:        uuid.New().String(),
		socket:    sock,
		closed:    make(chan bool),
		sendQueue: make(chan *Message, 128),
		recvQueue: make(chan *Message, 128),
	}
	go l.process()
	return l
}

// link methods

// process processes messages on the send queue.
// these are messages to be sent to the remote side.
func (l *link) process() {
	go func() {
		for {
			m := new(Message)
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
func (l *link) send(m *Message) error {
	tm := new(transport.Message)
	tm.Header = m.Header
	tm.Body = m.Body
	// send via the transport socket
	return l.socket.Send(tm)
}

// recv a message on the link
func (l *link) recv(m *Message) error {
	if m.Header == nil {
		m.Header = make(map[string]string)
	}

	tm := new(transport.Message)

	// receive the transport message
	if err := l.socket.Recv(tm); err != nil {
		return err
	}

	// set the message
	m.Header = tm.Header
	m.Body = tm.Body

	return nil
}

// Close the link
func (l *link) Close() error {
	select {
	case <-l.closed:
		return nil
	default:
		close(l.closed)
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
func (l *link) Recv(m *Message) error {
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
func (l *link) Send(m *Message) error {
	select {
	case <-l.closed:
		return io.EOF
	case l.sendQueue <- m:
	}
	return nil
}
