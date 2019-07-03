package network

import (
	"errors"
	"io"
	"sync"

	gproto "github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/codec/proto"
	pb "github.com/micro/go-micro/network/proto"
	"github.com/micro/go-micro/transport"
)

type link struct {
	// the embedded node
	*node

	closed chan bool

	sync.RWMutex

	// the link id
	id string

	// the send queue to the socket
	sendQueue chan *Message
	// the recv queue to the socket
	recvQueue chan *Message

	// codec we use to marshal things
	codec codec.Marshaler

	// the socket for this link
	socket transport.Socket

	// the lease for this link
	lease *pb.Lease

	// determines the cost of the link
	// based on queue length and roundtrip
	length int
	weight int
}

var (
	ErrLinkClosed = errors.New("link closed")
)

func newLink(n *node, sock transport.Socket, lease *pb.Lease) *link {
	return &link{
		id:        uuid.New().String(),
		closed:    make(chan bool),
		codec:     &proto.Marshaler{},
		node:      n,
		lease:     lease,
		socket:    sock,
		sendQueue: make(chan *Message, 128),
		recvQueue: make(chan *Message, 128),
	}
}

// link methods

// process processes messages on the send queue.
// these are messages to be sent to the remote side.
func (l *link) process() {
	go func() {
		for {
			m := new(Message)
			if err := l.recv(m, nil); err != nil {
				return
			}

			// check if it's an internal close method
			if m.Header["Micro-Method"] == "close" {
				l.Close()
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
			if err := l.send(m, nil); err != nil {
				return
			}
		case <-l.closed:
			return
		}
	}
}

// accept waits for the connect message from the remote end
// if it receives anything else it throws an error
func (l *link) accept() error {
	for {
		m := new(transport.Message)
		err := l.socket.Recv(m)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// TODO: pick a reliable header
		event := m.Header["Micro-Method"]

		switch event {
		// connect event
		case "connect":
			// process connect events from network.Connect()
			// these are new connections to join the network

			// decode the connection event
			conn := new(pb.Connect)
			// expecting a connect message
			if err := l.codec.Unmarshal(m.Body, conn); err != nil {
				// skip error
				continue
			}

			// no micro id close the link
			if len(conn.Muid) == 0 {
				l.Close()
				return errors.New("invalid muid " + conn.Muid)
			}

			// get the existing lease if it exists
			lease := conn.Lease
			// if there's no lease create a new one
			if lease == nil {
				// create a new lease/node
				lease = l.node.network.lease(conn.Muid)
			}

			// check if we connected to ourself
			if conn.Muid == l.node.muid {
				// check our own leasae
				l.node.Lock()
				if l.node.lease == nil {
					l.node.lease = lease
				}
				l.node.Unlock()
			}

			// set the author to our own muid
			lease.Author = l.node.muid

			// send back a lease offer for the node
			if err := l.send(&Message{
				Header: map[string]string{
					"Micro-Method": "lease",
				},
			}, lease); err != nil {
				return err
			}

			// the lease is saved
			l.Lock()
			l.lease = lease
			l.Unlock()

			// we've connected
			// start processing the messages
			go l.process()
			return nil
		case "close":
			l.Close()
			return io.EOF
		default:
			return errors.New("unknown method: " + event)
		}
	}
}

// connect sends a connect request and waits on a lease.
// this is for a new connection. in the event we send
// an existing lease, the same lease should be returned.
// if it differs then we assume our address for this link
// is different...
func (l *link) connect() error {
	// get the current lease
	l.RLock()
	lease := l.lease
	l.RUnlock()

	// send a lease request
	if err := l.send(&Message{
		Header: map[string]string{
			"Micro-Method": "connect",
		},
	}, &pb.Connect{Muid: l.node.muid, Lease: lease}); err != nil {
		return err
	}

	// create the new things
	tm := new(Message)
	newLease := new(pb.Lease)

	// wait for a response, hopefully a lease
	if err := l.recv(tm, newLease); err != nil {
		return err
	}

	event := tm.Header["Micro-Method"]

	// check the method
	switch event {
	case "lease":
		// save the lease
		l.Lock()
		l.lease = newLease
		l.Unlock()

		// start processing the messages
		go l.process()
	case "close":
		l.Close()
		return io.EOF
	default:
		l.Close()
		return errors.New("unable to attain lease")
	}

	return nil
}

// send a message over the link
func (l *link) send(m *Message, v interface{}) error {
	tm := new(transport.Message)
	tm.Header = m.Header
	tm.Body = m.Body

	// set the body if not nil
	// we're assuming this is network message
	if v != nil {
		// encode the data
		b, err := l.codec.Marshal(v)
		if err != nil {
			return err
		}

		// set the content type
		tm.Header["Content-Type"] = "application/protobuf"
		// set the marshalled body
		tm.Body = b
	}

	// send via the transport socket
	return l.socket.Send(tm)
}

// recv a message on the link
func (l *link) recv(m *Message, v interface{}) error {
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

	// bail early
	if v == nil {
		return nil
	}

	// try unmarshal the body
	// skip if there's no content-type
	if tm.Header["Content-Type"] != "application/protobuf" {
		return nil
	}

	// return unmarshalled
	return l.codec.Unmarshal(m.Body, v.(gproto.Message))
}

// Close the link
func (l *link) Close() error {
	select {
	case <-l.closed:
		return nil
	default:
		close(l.closed)
	}

	// send a final close message
	return l.socket.Send(&transport.Message{
		Header: map[string]string{
			"Micro-Method": "close",
		},
	})
}

// returns the node id
func (l *link) Id() string {
	l.RLock()
	defer l.RUnlock()
	if l.lease == nil {
		return ""
	}
	return l.lease.Node.Id
}

// Address of the node we're connected to
func (l *link) Address() string {
	l.RLock()
	defer l.RUnlock()
	if l.lease == nil {
		return l.socket.Remote()
	}
	// the node in the lease
	return l.lease.Node.Address
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
func (l *link) Accept() (*Message, error) {
	select {
	case <-l.closed:
		return nil, io.EOF
	case m := <-l.recvQueue:
		return m, nil
	}
	// never reach
	return nil, nil
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
