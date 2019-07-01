package network

import (
	"crypto/sha256"
	"fmt"
	"io"
	"runtime/debug"
	"sync"
	"time"

	gproto "github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/codec/proto"
	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/network/proxy"
	"github.com/micro/go-micro/network/resolver"
	"github.com/micro/go-micro/network/router"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/transport"
	"github.com/micro/go-micro/util/log"

	pb "github.com/micro/go-micro/network/proto"
	nreg "github.com/micro/go-micro/network/resolver/registry"
)

type network struct {
	options.Options

	// resolver use to connect to the network
	resolver resolver.Resolver

	// router used to find routes in the network
	router router.Router

	// proxy used to route through the network
	proxy proxy.Proxy

	// id of this network
	id string

	// links maintained for this network
	// based on peers not nodes. maybe maintain
	// node separately or note that links have nodes
	mtx   sync.RWMutex
	links []Link
}

type node struct {
	*network

	// closed channel
	closed chan bool

	mtx sync.RWMutex

	// the node id
	id string

	// address of this node
	address string

	// the node registry
	registry registry.Registry

	// the base level transport
	transport transport.Transport

	// the listener
	listener transport.Listener

	// leases for connections to us
	// link id:link
	links map[string]*link
}

type link struct {
	// the embedded node
	*node

	// the link id
	id string

	// queue buffer for this link
	queue chan *Message

	// the socket for this link
	socket *socket

	// the lease for this link
	lease *pb.Lease

	// length and weight of the link
	mtx sync.RWMutex

	// determines the cost of the link
	// based on queue length and roundtrip
	length int
	weight int
}

type socket struct {
	node   *node
	codec  codec.Marshaler
	socket transport.Socket
}

// network methods

// lease generates a new lease with a node id/address
// TODO: use a consensus mechanism, pool or some deterministic
// unique prefixing method.
func (n *network) lease() *pb.Lease {
	// create the id
	id := uuid.New().String()
	// create a timestamp
	now := time.Now().UnixNano()
	// create the address
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s-%d\n", id, now)))
	address := fmt.Sprintf("%x", h.Sum(nil))

	// return the node
	return &pb.Lease{
		Id:        id,
		Timestamp: now,
		Node: &pb.Node{
			Id:      id,
			Address: address,
		},
	}
}

// lookup returns a list of network records in priority order of local
func (n *network) lookup(r registry.Registry) []*resolver.Record {
	// create a registry resolver to find local nodes
	rr := nreg.Resolver{Registry: r}

	// get all the nodes for the network that are local
	localRecords, err := rr.Resolve("network:" + n.Id())
	if err != nil {
		// we're not in a good place here
	}

	// if its a local network we never try lookup anything else
	if n.Id() == "local" {
		return localRecords
	}

	// now resolve incrementally based on resolvers specified
	networkRecords, err := n.resolver.Resolve(n.Id())
	if err != nil {
		// still not in a good place
	}

	// return aggregate records
	return append(localRecords, networkRecords...)
}

func (n *network) Id() string {
	return n.id
}

// Connect connects to the network and returns a new node.
// The node is the callers connection to the network. They
// should advertise this address to people. Anyone else
// on the network should be able to route to it.
func (n *network) Connect() (Node, error) {
	// create a new node
	node := new(node)
	// closed channel
	node.closed = make(chan bool)
	// set the nodes network
	node.network = n

	// initially we have no id
	// create an id and address
	// TODO: create a real unique id and address
	// lease := n.lease()
	// set the node id
	// node.id = lease.Node.Id

	// get the transport we're going to use for our tunnels
	t, ok := n.Options.Values().Get("network.transport")
	if ok {
		node.transport = t.(transport.Transport)
	} else {
		// TODO: set to quic
		node.transport = transport.DefaultTransport
	}

	// start the node

	// we listen on a random address, this is not advertised
	// TODO: use util/addr to get something anyone in the same private network can talk to
	l, err := node.transport.Listen(":0")
	if err != nil {
		return nil, err
	}
	// set the listener
	node.listener = l

	// TODO: this should be an overlay address
	// ideally received via some dhcp style broadcast
	node.address = l.Addr()

	// TODO: start the router and broadcast advertisements
	// receive updates and push them to the network in accept(l) below
	// chan, err := n.router.Advertise()
	// u <- chan
	// socket.send("route", u)
	// u := socket.recv() => r.router.Update(u)

	// process any incoming messages on the listener
	// this is our inbound network connection
	node.accept(l)

	// register the node with the registry for the network
	// TODO: use a registrar or something else for local things
	r, ok := n.Options.Values().Get("network.registry")
	if ok {
		node.registry = r.(registry.Registry)
	} else {
		node.registry = registry.DefaultRegistry
	}

	// lookup the network to see if there's any nodes
	records := n.lookup(node.registry)

	// should we actually do this?
	if len(records) == 0 {
		// set your own node id
		lease := n.lease()
		node.id = lease.Node.Id
	}

	// register self with the network registry
	// this is a local registry of nodes separate to the resolver
	// maybe consolidate registry/resolver
	// TODO: find a way to do this via gossip or something else
	if err := node.registry.Register(&registry.Service{
		// register with the network id
		Name: "network:" + n.Id(),
		Nodes: []*registry.Node{
			{Id: node.id, Address: node.address},
		},
	}); err != nil {
		node.Close()
		return nil, err
	}

	// create a channel to get links
	linkChan := make(chan *link, 1)

	// we're going to wait for the first connection
	go node.connect(linkChan)

	// wait forever to connect
	// TODO: do something with the links we receive
	<-linkChan

	return node, nil
}

// TODO: establish links for peering networks
func (n *network) Peer(Network) (Link, error) {
	// New network was created using NewNetwork after receiving routes from a different node

	// Connect to the new network and be assigned a node

	// Transfer data between the networks

	// take other resolver
	// order: registry (local), ...resolver
	// resolve the network

	// periodically connect to nodes resolved in the network
	// and add to the network links
	return nil, nil
}

func (n *network) Links() ([]Link, error) {
	n.mtx.RLock()
	defer n.mtx.RUnlock()
	return n.links, nil
}

// node methods

// Accept processes the incoming messages on its listener.
// This listener was created with the first call to network.Connect.
// Any inbound new socket here is essentially something else attempting
// to connect to the network. So we turn it into a socket, then process it.
func (n *node) accept(l transport.Listener) error {
	return l.Accept(func(sock transport.Socket) {
		defer func() {
			// close socket
			sock.Close()

			if r := recover(); r != nil {
				log.Log("panic recovered: ", r)
				log.Log(string(debug.Stack()))
			}
		}()

		// create a new link
		// generate a new link
		link := &link{
			node: n,
			id:   uuid.New().String(),
		}
		// create a new network socket
		sk := new(socket)
		sk.node = n
		sk.codec = proto.Marshaler{}
		sk.socket = sock

		// set link socket
		link.socket = sk

		// accept messages on the socket
		// blocks forever or until error
		if err := link.up(); err != nil {
			// TODO: delete link
		}
	})
}

// connect attempts to periodically connect to new nodes in the network.
// It will only do this if it has less than 3 connections. this method
// is called by network.Connect and fired in a go routine after establishing
// the first connection and creating a node. The node attempts to maintain
// its connection to the network via multiple links.
func (n *node) connect(linkChan chan *link) {
	// TODO: adjustable ticker
	t := time.NewTicker(time.Second)
	var lease *pb.Lease

	for {
		select {
		// on every tick check the number of links and then attempt
		// to connect to new nodes if we don't have sufficient links
		case <-t.C:
			n.mtx.RLock()

			// only start processing if we have less than 3 links
			if len(n.links) > 2 {
				n.mtx.RUnlock()
				continue
			}

			// get a list of link addresses so we don't reconnect
			// to the ones we're already connected to
			nodes := map[string]bool{}
			for _, l := range n.links {
				nodes[l.lease.Node.Address] = true
			}

			n.mtx.RUnlock()

			records := n.network.lookup(n.registry)

			// for each record check we haven't already got a connection
			// attempt to dial it, create a new socket and call
			// connect with our existing network lease.
			// if its the first call we don't actually have a lease

			// TODO: determine how to prioritise local records
			// while still connecting to the global network
			for _, record := range records {
				// skip existing connections
				if nodes[record.Address] {
					continue
				}

				// attempt to connect and create a link

				// connect to the node
				s, err := n.transport.Dial(record.Address)
				if err != nil {
					continue
				}

				// create a new socket
				sk := &socket{
					node:   n,
					codec:  &proto.Marshaler{},
					socket: s,
				}

				// broadcast a "connect" request and get back "lease"
				// this is your tunnel to the outside world and to the network
				// then push updates and messages over this link
				// first connect will not have a lease so we get one with node id/address
				l, err := sk.connect(lease)
				if err != nil {
					s.Close()
					continue
				}

				// set lease for next time
				lease = l

				// create a new link with the lease and socket
				link := &link{
					id:     uuid.New().String(),
					lease:  lease,
					node:   n,
					queue:  make(chan *Message, 128),
					socket: sk,
				}

				// bring up the link
				go link.up()

				// save the new link
				n.mtx.Lock()
				n.links[link.id] = link
				n.mtx.Unlock()

				// drop this down the link channel to the network
				// so it can manage the links
				select {
				case linkChan <- link:
				// we don't wait for anyone
				default:
				}
			}
		case <-n.closed:
			return
		}
	}
}

func (n *node) Address() string {
	return n.address
}

// Close shutdowns all the links and closes the listener
func (n *node) Close() error {
	select {
	case <-n.closed:
		return nil
	default:
		close(n.closed)
		// shutdown all the links
		n.mtx.Lock()
		for id, link := range n.links {
			link.down()
			delete(n.links, id)
		}
		n.mtx.Unlock()
		// deregister self
		n.registry.Deregister(&registry.Service{
			Name: "network:" + n.network.Id(),
			Nodes: []*registry.Node{
				{Id: n.id, Address: n.address},
			},
		})
		return n.listener.Close()
	}
	return nil
}

func (n *node) Accept() (*Message, error) {
	// process the inbound cruft

	return nil, nil
}

func (n *node) Links() ([]Link, error) {
	n.mtx.RLock()
	defer n.mtx.RUnlock()

	var links []Link
	for _, l := range n.links {
		links = append(links, l)
	}
	return links, nil
}

func (n *node) Network() Network {
	return n.network
}

func (n *node) Send(m *Message) error {
	n.mtx.RLock()
	defer n.mtx.RUnlock()

	var gerr error

	// send to all links
	// TODO: be smarter
	for _, link := range n.links {
		// TODO: process the error, do some link flap detection
		// blackhold the connection, etc
		if err := link.socket.send(m, nil); err != nil {
			gerr = err
			continue
		}
	}

	return gerr
}

// link methods

// bring up the link
func (l *link) up() error {
	// TODO: manage the length/weight of the link
	return l.socket.accept()
}

// kill the link
func (l *link) down() error {
	return l.socket.close()
}

func (l *link) Length() int {
	l.mtx.RLock()
	defer l.mtx.RUnlock()
	return l.length
}

func (l *link) Weight() int {
	l.mtx.RLock()
	defer l.mtx.RUnlock()
	return l.weight
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

func (s *socket) close() error {
	return s.socket.Close()
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
