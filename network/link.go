package network

import (
	"errors"
	"io"
	"sync"

	"github.com/micro/go-micro/util/log"
	gproto "github.com/golang/protobuf/proto"
	"github.com/micro/go-micro/codec"
	pb "github.com/micro/go-micro/network/proto"
	"github.com/micro/go-micro/transport"
)

type link struct {
	// the embedded node
	*node

	sync.RWMutex

	// the link id
	id string

	// the send queue to the socket
	queue chan *Message

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

// link methods

// process processe messages on the send queue
func (l *link) process() {
	for {
		select {
		case m := <-l.queue:
			if err := l.send(m, nil); err != nil {
				return
			}
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
		case "Connect":
			// process connect events from network.Connect()
			// these are new connections to join the network

			// decode the connection event
			conn := new(pb.Connect)
			if err := l.codec.Unmarshal(m.Body, conn); err != nil {
				// skip error
				continue
			}

			// get the existing lease if it exists
			lease := conn.Lease
			// if there's no lease create a new one
			if lease == nil {
				// create a new lease/node
				lease = l.node.network.lease()
			}

			// send back a lease offer for the node
			if err := l.send(&Message{
				Header: map[string]string{
					"Micro-Method": "Lease",
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
		case "Close":
			l.Close()
			return errors.New("connection closed")
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
			"Micro-Method": "Connect",
		},
	}, &pb.Connect{Lease: lease}); err != nil {
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
	case "Lease":
		// save the lease
		l.Lock()
		l.lease = newLease
		l.Unlock()

		// start processing the messages
		go l.process()
	case "Close":
		l.socket.Close()
		return errors.New("connection closed")
	default:
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

	log.Debugf("link %s sending %+v %+v\n", l.id, m, v)

	// send via the transport socket
	return l.socket.Send(&transport.Message{
		Header: m.Header,
		Body:   m.Body,
	})
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

	log.Debugf("link %s receiving %+v %+v\n", l.id, tm, v)

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
	// send a final close message
	l.socket.Send(&transport.Message{
		Header: map[string]string{
			"Micro-Method": "Close",
		},
	})
	// close the socket
	return l.socket.Close()
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
	l.RLock()
	defer l.RUnlock()
	return l.weight
}

func (l *link) Accept() (*Message, error) {
	m := new(Message)
	err := l.recv(m, nil)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (l *link) Send(m *Message) error {
	return l.send(m, nil)
}
