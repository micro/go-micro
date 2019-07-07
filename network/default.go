package network

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/network/proxy"
	"github.com/micro/go-micro/network/proxy/mucp"
	"github.com/micro/go-micro/network/resolver"
	"github.com/micro/go-micro/network/router"
	"github.com/micro/go-micro/registry"

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

	// name of this network
	name string

	// links maintained for this network
	// based on peers not nodes. maybe maintain
	// node separately or note that links have nodes
	mtx   sync.RWMutex
	links []Link
}

// network methods

// lease generates a new lease with a node id/address
// TODO: use a consensus mechanism, pool or some deterministic
// unique addressing method.
func (n *network) lease(muid string) *pb.Lease {
	// create the id
	id := uuid.New().String()
	// create a timestamp
	now := time.Now().UnixNano()

	// create the address by hashing the id and timestamp
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s-%d\n", id, now)))
	// magic new address
	address := fmt.Sprintf("%x", h.Sum(nil))

	// return the node
	return &pb.Lease{
		Id:        id,
		Timestamp: now,
		Node: &pb.Node{
			Muid:    muid,
			Id:      id,
			Address: address,
			Network: n.name,
		},
	}
}

// lookup returns a list of network records in priority order of local
func (n *network) lookup(r registry.Registry) []*resolver.Record {
	// create a registry resolver to find local nodes
	rr := nreg.Resolver{Registry: r}

	// get all the nodes for the network that are local
	localRecords, err := rr.Resolve(n.Name())
	if err != nil {
		// we're not in a good place here
	}

	// if its a local network we never try lookup anything else
	if n.Name() == "local" {
		return localRecords
	}

	// now resolve incrementally based on resolvers specified
	networkRecords, err := n.resolver.Resolve(n.Name())
	if err != nil {
		// still not in a good place
	}

	// return aggregate records
	return append(localRecords, networkRecords...)
}

func (n *network) Name() string {
	return n.name
}

// Connect connects to the network and returns a new node.
// The node is the callers connection to the network. They
// should advertise this address to people. Anyone else
// on the network should be able to route to it.
func (n *network) Connect() (Node, error) {
	return newNode(n)
}

// Peer is used to establish a link between two networks.
// e.g micro.mu connects to example.com and share routes
// This is done by creating a new node on both networks
// and creating a link between them.
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

// newNetwork returns a new network interface
func newNetwork(opts ...options.Option) *network {
	options := options.NewOptions(opts...)

	// new network instance with defaults
	net := &network{
		Options:  options,
		name:     DefaultName,
		router:   router.DefaultRouter,
		proxy:    new(mucp.Proxy),
		resolver: new(nreg.Resolver),
	}

	// get network name
	name, ok := options.Values().Get("network.name")
	if ok {
		net.name = name.(string)
	}

	// get router
	r, ok := options.Values().Get("network.router")
	if ok {
		net.router = r.(router.Router)
	}

	// get proxy
	p, ok := options.Values().Get("network.proxy")
	if ok {
		net.proxy = p.(proxy.Proxy)
	}

	// get resolver
	res, ok := options.Values().Get("network.resolver")
	if ok {
		net.resolver = res.(resolver.Resolver)
	}

	return net
}
