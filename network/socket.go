package network

import (
	"io"

	gproto "github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/transport"

	pb "github.com/micro/go-micro/network/proto"
)

type socket struct {
	node   *node
	codec  codec.Marshaler
	socket transport.Socket
}

func (s *socket) close() error {
	return s.socket.Close()
}

// accept is the state machine that processes messages on the socket
func (s *socket) accept() error {
	for {
		m := new(transport.Message)
		err := s.socket.Recv(m)
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
			if err := s.codec.Unmarshal(m.Body, conn); err != nil {
				// skip error
				continue
			}

			// get the existing lease if it exists
			lease := conn.Lease
			if lease == nil {
				// create a new lease/node
				lease = s.node.network.lease()
			}

			// send back a lease offer for the node
			if err := s.send(&Message{
				Header: map[string]string{
					"Micro-Method": "lease",
				},
			}, lease); err != nil {
				return err
			}

			// record this mapping of socket to node/lease
			s.node.mtx.Lock()
			id := uuid.New().String()
			s.node.links[id] = &link{
				node:   s.node,
				id:     id,
				lease:  lease,
				queue:  make(chan *Message, 128),
				socket: s,
			}
			s.node.mtx.Unlock()
		// a route update
		case "route":
			// process router events

		// received a lease
		case "lease":
		// no op as we don't process lease events on existing connections
		// these are in response to a connect message
		default:
			// process all other messages
		}
	}
}

// connect sends a connect request and waits on a lease.
// this is for a new connection. in the event we send
// an existing lease, the same lease should be returned.
// if it differs then we assume our address for this link
// is different...
func (s *socket) connect(l *pb.Lease) (*pb.Lease, error) {
	// send a lease request
	if err := s.send(&Message{
		Header: map[string]string{
			"Micro-Method": "connect",
		},
	}, &pb.Connect{Lease: l}); err != nil {
		return nil, err
	}

	// create the new things
	tm := new(Message)
	lease := new(pb.Lease)

	// wait for a lease response
	if err := s.recv(tm, lease); err != nil {
		return nil, err
	}

	return lease, nil
}

func (s *socket) send(m *Message, v interface{}) error {
	tm := new(transport.Message)
	tm.Header = m.Header
	tm.Body = m.Body

	// set the body if not nil
	// we're assuming this is network message
	if v != nil {
		// encode the data
		b, err := s.codec.Marshal(v)
		if err != nil {
			return err
		}

		// set the content type
		tm.Header["Content-Type"] = "application/protobuf"
		// set the marshalled body
		tm.Body = b
	}

	// send via the transport socket
	return s.socket.Send(&transport.Message{
		Header: m.Header,
		Body:   m.Body,
	})
}

func (s *socket) recv(m *Message, v interface{}) error {
	if m.Header == nil {
		m.Header = make(map[string]string)
	}

	tm := new(transport.Message)

	// receive the transport message
	if err := s.socket.Recv(tm); err != nil {
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
	return s.codec.Unmarshal(m.Body, v.(gproto.Message))
}
