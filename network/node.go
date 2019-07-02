package network

import (
	"errors"
	"fmt"
	"net"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/codec/proto"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/transport"
	"github.com/micro/go-micro/util/addr"
	"github.com/micro/go-micro/util/log"

	pb "github.com/micro/go-micro/network/proto"
)

type node struct {
	*network

	// closed channel
	closed chan bool

	sync.RWMutex

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

	// messages received over links
	recv chan *Message
	// messages received over links
	send chan *Message
}

// network methods

func newNode(n *network) (*node, error) {
	// create a new node
	node := &node{
		// the links
		links: make(map[string]*link),
		// closed channel
		closed: make(chan bool),
		// set the nodes network
		network: n,
		// set the default transport
		transport: transport.DefaultTransport,
		// set the default registry
		registry: registry.DefaultRegistry,
		// receive channel for accepted connections
		recv: make(chan *Message, 128),
		// send channel for accepted connections
		send: make(chan *Message, 128),
	}

	// get the transport we're going to use for our tunnels
	// TODO: set to quic or tunnel or something else
	t, ok := n.Options.Values().Get("network.transport")
	if ok {
		node.transport = t.(transport.Transport)
	}

	// register the node with the registry for the network
	// TODO: use a registrar or something else for local things
	r, ok := n.Options.Values().Get("network.registry")
	if ok {
		node.registry = r.(registry.Registry)
	}

	// we listen on a random address, this is not advertised
	// TODO: use util/addr to get something anyone in the same private network can talk to
	l, err := node.transport.Listen(":0")
	if err != nil {
		return nil, err
	}
	// set the listener
	node.listener = l

	node.address = l.Addr()

	// TODO: start the router and broadcast advertisements
	// receive updates and push them to the network in accept(l) below
	// chan, err := n.router.Advertise()
	// u <- chan
	// socket.send("route", u)
	// u := socket.recv() => r.router.Update(u)

	// process any incoming messages on the listener
	// this is our inbound network connection
	go node.accept(l)

	// process any messages being sent by node.Send
	// forwards to every link we have
	go node.process()

	// lookup the network to see if there's any nodes
	records := n.lookup(node.registry)

	// assuming if there are no records, we are the first
	// we set ourselves a lease. should we actually do this?
	if len(records) == 0 {
		// set your own node id
		lease := n.lease()
		node.id = lease.Node.Id
	}

	var port int
	// TODO: this should be an overlay address
	// ideally received via some dhcp style broadcast
	host, pp, err := net.SplitHostPort(l.Addr())
	if err == nil {
		pt, _ := strconv.Atoi(pp)
		port = pt
	}

	// some horrible things are happening
	if host == "::" {
		host = ""
	}
	// set the address
	addr, _ := addr.Extract(host)

	node.address = fmt.Sprintf("%s:%d", addr, port)

	// register self with the registry using network: prefix
	// this is a local registry of nodes separate to the resolver
	// maybe consolidate registry/resolver
	// TODO: find a way to do this via gossip or something like
	// a registrar or tld or whatever
	if err := node.registry.Register(&registry.Service{
		// register with the network id
		Name: "network:" + n.Id(),
		Nodes: []*registry.Node{
			{Id: node.id, Address: addr, Port: port},
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
	link := <-linkChan

	// process this link
	go node.manage(link)

	go func() {
		// process any further new links
		select {
		case l := <-linkChan:
			go node.manage(l)
		case <-node.closed:
			return
		}
	}()

	return node, nil
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
		link := &link{
			// link has a unique id
			id: uuid.New().String(),
			// proto marshaler
			codec: proto.Marshaler{},
			// link has a socket
			socket: sock,
			// for generating leases,
			node: n,
			// the send queue,
			queue: make(chan *Message, 128),
		}

		log.Debugf("Accepting connection from %s", link.socket.Remote())

		// wait for the link to be connected
		// the remote end will send "Connect"
		// and we will return a "Lease"
		if err := link.accept(); err != nil {
			return
		}

		log.Debugf("Accepted link from %s", link.socket.Remote())

		// save with the remote address as the key
		// where we attempt to connect to nodes
		// we do not connect to the same thing
		n.Lock()
		n.links[link.socket.Remote()] = link
		n.Unlock()

		// manage the link for its lifetime
		n.manage(link)
	})
}

// processes the send queue
func (n *node) process() {
	for {
		select {
		case <-n.closed:
			return
		// process outbound messages on the send queue
		// these messages are received from n.Send
		case m := <-n.send:
			// queue the message on each link
			// TODO: more than likely use proxy
			n.RLock()
			for _, l := range n.links {
				l.queue <- m
			}
			n.RUnlock()
		}
	}
}

func (n *node) manage(l *link) {
	// now process inbound messages on the link
	// assumption is this handles everything else
	for {
		// get a message on the link
		m := new(Message)
		if err := l.recv(m, nil); err != nil {
			// ???
			return
		}

		select {
		case <-n.closed:
			return
		// send to the recv channel e.g node.Accept()
		case n.recv <- m:
		}
	}
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
		// exit when told to do so
		case <-n.closed:
			return
		// on every tick check the number of links and then attempt
		// to connect to new nodes if we don't have sufficient links
		case <-t.C:
			n.RLock()

			// only start processing if we have less than 3 links
			if len(n.links) > 2 {
				n.RUnlock()
				continue
			}

			// get a list of link addresses so we don't reconnect
			// to the ones we're already connected to
			nodes := map[string]bool{}
			for addr, _ := range n.links {
				// id is the lookup address used to connect
				nodes[addr] = true
			}

			// unlock our read lock
			n.RUnlock()

			// lookup records for our network
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
					log.Debugf("Skipping connection to %s", record.Address)
					continue
				}

				// attempt to connect and create a link

				log.Debugf("Dialing connection to %s", record.Address)

				// connect to the node
				sock, err := n.transport.Dial(record.Address)
				if err != nil {
					log.Debugf("Dialing connection error %v", err)
					continue
				}

				// create a new link with the lease and socket
				link := &link{
					codec:  &proto.Marshaler{},
					id:     uuid.New().String(),
					lease:  lease,
					socket: sock,
					queue:  make(chan *Message, 128),
				}

				log.Debugf("Connecting link to %s", record.Address)

				// connect the link:
				// this broadcasts a "connect" request and gets back a "lease"
				// this is the tunnel to the outside world and to the network
				// then push updates and messages over this link
				// first connect will not have a lease so we get one with node id/address
				if err := link.connect(); err != nil {
					// shit
					link.Close()
					continue
				}

				log.Debugf("Connected link to %s", record.Address)

				n.Lock()
				// set lease for next time we connect to anything else
				// we want to use the same lease for that. in future
				// we may have to expire the lease
				lease = link.lease
				// save the new link
				n.links[link.socket.Remote()] = link
				n.Unlock()

				// drop this down the link channel to the network
				// so it can manage the links
				select {
				case linkChan <- link:
				// we don't wait for anyone
				default:
				}
			}
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
		// mark as closed
		close(n.closed)

		// shutdown all the links
		n.Lock()
		for id, link := range n.links {
			link.Close()
			delete(n.links, id)
		}
		n.Unlock()

		// deregister self
		n.registry.Deregister(&registry.Service{
			Name: "network:" + n.network.Id(),
			Nodes: []*registry.Node{
				{Id: n.id, Address: n.address},
			},
		})

		// shutdown the listener
		return n.listener.Close()
	}
	return nil
}

// Accept receives the incoming messages from all links
func (n *node) Accept() (*Message, error) {
	// process the inbound cruft
	for {
		select {
		case m, ok := <-n.recv:
			if !ok {
				return nil, errors.New("connection closed")
			}
			// return the message
			return m, nil
		case <-n.closed:
			return nil, errors.New("connection closed")
		}
	}
	// we never get here
	return nil, nil
}

func (n *node) Network() string {
	return n.network.id
}

// Send propagates a message over all links. This should probably use its proxy.
func (n *node) Send(m *Message) error {
	select {
	case <-n.closed:
		return errors.New("connection closed")
	case n.send <- m:
		// send the message
	}
	return nil
}
