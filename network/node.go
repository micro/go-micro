package network

import (
	"errors"
	"fmt"
	"io"
	"net"
	"runtime/debug"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/transport"
	"github.com/micro/go-micro/util/addr"
	"github.com/micro/go-micro/util/log"

	pb "github.com/micro/go-micro/network/proto"
)

type node struct {
	*network

	// closed channel to close our connection to the network
	closed chan bool

	sync.RWMutex

	// the nodes unique micro assigned mac address
	muid string

	// the node id registered in registry
	id string

	// address of this node registered in registry
	address string

	// our network lease with our network id/address
	lease *pb.Lease

	// the node registry
	registry registry.Registry

	// the base level transport
	transport transport.Transport

	// the listener
	listener transport.Listener

	// connected records
	// record.Address:true
	connected map[string]bool

	// leases for connections to us
	// link remote node:link
	links map[string][]*link

	// messages received over links
	recv chan *Message
	// messages received over links
	send chan *Message
}

// network methods

func newNode(n *network) (*node, error) {
	// create a new node
	node := &node{
		// this nodes unique micro assigned mac address
		muid: fmt.Sprintf("%s-%s", n.id, uuid.New().String()),
		// map of connected records
		connected: make(map[string]bool),
		// the links
		links: make(map[string][]*link),
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

	// used to register in registry for network resolution
	// separate to our lease on the network itself
	node.id = uuid.New().String()
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
	log.Debugf("connect managing link %s", link.id)
	go node.manage(link)

	go func() {
		for {
			// process any further new links
			select {
			case l := <-linkChan:
				log.Debugf("Managing new link %s", l.id)
				go node.manage(l)
			case <-node.closed:
				return
			}
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
		link := newLink(n, sock, nil)

		log.Debugf("Accepting connection from %s", link.socket.Remote())

		// wait for the link to be connected
		// the remote end will send "Connect"
		// and we will return a "Lease"
		if err := link.accept(); err != nil {
			log.Debugf("Error accepting connection %v", err)
			return
		}

		log.Debugf("Accepted link from %s", link.socket.Remote())

		// save with the muid as the key
		// where we attempt to connect to nodes
		// we do not connect to the same thing

		// TODO: figure out why this is an issue
		// When we receive a connection from ourself
		// we can't maintain the two links separately
		// so we don't save it. It's basically some
		// weird loopback issue because its our own socket.
		if n.muid != link.lease.Node.Muid {
			n.Lock()
			// get the links

			links := n.links[link.lease.Node.Muid]
			// append to the current links
			links = append(links, link)
			// save the links with muid as the key
			n.links[link.lease.Node.Muid] = links
			n.Unlock()
		}

		// manage the link for its lifetime
		log.Debugf("managing the link now %s", link.id)
		n.manage(link)
	})
}

// processes the sends the messages from n.Send into the queue of
// each link. If multiple links exist for a muid it should only
// send on link to figure it out.
// If we connected to a record and that link goes down we should
// also remove it from the n.connected map.
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
			// range over all the links
			for _, links := range n.links {
				if len(links) == 0 {
					continue
				}

				// sort the links by weight
				sort.Slice(links, func(i, j int) bool {
					return links[i].Weight() < links[j].Weight()
				})

				// queue the message
				links[0].Send(m)
			}
			n.RUnlock()
		}
	}
}

// Manage manages the link for its lifetime. It should ideally throw
// away the link in the n.links map if there's any issues or total disconnection
// it should look at link.Status.
// If we connected to a record and that link goes down we should
// also remove it from the n.connected map.
func (n *node) manage(l *link) {
	// now process inbound messages on the link
	// assumption is this handles everything else
	for {
		// the send side uses a link queue but the receive side immediately sends it
		// ideally we should probably have an internal queue on that side as well
		// so we can judge link saturation both ways.

		m, err := l.Accept()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Debugf("Error accepting message on link %s: %v", l.id, err)
			// ???
			return
		}

		// if the node connection is closed bail out
		select {
		case <-n.closed:
			return
		// send to the network recv channel e.g node.Accept()
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
			conns := len(n.links)
			if conns > 2 {
				n.RUnlock()
				continue
			}

			// get a list of link addresses so we don't reconnect
			// to the ones we're already connected to
			connected := n.connected

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
				if connected[record.Address] {
					log.Tracef("Skipping connection to %s", record.Address)
					continue
				}

				// check how many connections we have
				if conns > 2 {
					log.Debugf("Made enough connections")
					break
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
				link := newLink(n, sock, lease)

				log.Debugf("Connecting link to %s", record.Address)

				// connect the link:
				// this broadcasts a "connect" request and gets back a "lease"
				// this is the tunnel to the outside world and to the network
				// then push updates and messages over this link
				// first connect will not have a lease so we get one with node id/address
				if err := link.connect(); err != nil {
					// shit
					continue
				}

				log.Debugf("Connected link to %s", record.Address)

				n.Lock()
				// set lease for next time we connect to anything else
				// we want to use the same lease for that. in future
				// we may have to expire the lease
				lease = link.lease
				// save the new link
				// get existing links using the lease author
				links := n.links[lease.Author]
				// append to the links
				links = append(links, link)
				// save the links using the author
				n.links[lease.Author] = links
				n.Unlock()

				// update number of connections
				conns++

				// save the connection
				n.Lock()
				n.connected[record.Address] = true
				n.Unlock()

				// drop this down the link channel to the network
				// so it can manage the links
				linkChan <- link
			}
		}
	}
}

func (n *node) Address() string {
	n.RLock()
	defer n.RUnlock()
	// we have no address yet
	if n.lease == nil {
		return ""
	}
	// return node address in the lease
	return n.lease.Node.Address
}

// Close shutdowns all the links and closes the listener
func (n *node) Close() error {
	select {
	case <-n.closed:
		return nil
	default:
		// mark as closed, we're now useless and there's no coming back
		close(n.closed)

		// shutdown all the links
		n.Lock()
		for muid, links := range n.links {
			for _, link := range links {
				link.Close()
			}
			delete(n.links, muid)
		}
		// reset connected
		n.connected = nil
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

func (n *node) Id() string {
	n.RLock()
	defer n.RUnlock()
	if n.lease == nil {
		return ""
	}
	return n.lease.Node.Id
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
