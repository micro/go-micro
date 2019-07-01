// Package network is a package for defining a network overlay
package network

import (
	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/network/proxy"
	"github.com/micro/go-micro/network/proxy/mucp"
	"github.com/micro/go-micro/network/resolver"
	"github.com/micro/go-micro/network/resolver/registry"
	"github.com/micro/go-micro/network/router"
)

// Network defines a network interface. The network is a single
// shared network between all nodes connected to it. The network
// is responsible for routing messages to the correct services.
type Network interface {
	options.Options
	// Id of the network
	Id() string
	// Connect to the network
	Connect() (Node, error)
	// Peer with a neighboring network
	Peer(Network) (Link, error)
}

// Node represents a single node on a network
type Node interface {
	// Id of the node
	Id() string
	// Address of the node
	Address() string
	// The network of the node
	Network() string
	// Close the network connection
	Close() error
	// Accept messages on the network
	Accept() (*Message, error)
	// Send a message to the network
	Send(*Message) error
}

// Link is a connection between one network and another
type Link interface {
	// remote node the link is peered with
	Node
	// length defines the speed or distance of the link
	Length() int
	// weight defines the saturation or usage of the link
	Weight() int
}

// Message is the base type for opaque data
type Message struct {
	// Headers which provide local/remote info
	Header map[string]string
	// The opaque data being sent
	Body []byte
}

var (
	// The default network ID is local
	DefaultId = "local"

	// just the standard network element
	DefaultNetwork = NewNetwork()
)

// NewNetwork returns a new network interface
func NewNetwork(opts ...options.Option) Network {
	options := options.NewOptions(opts...)

	// new network instance
	net := &network{
		id: DefaultId,
	}

	// get network id
	id, ok := options.Values().Get("network.id")
	if ok {
		net.id = id.(string)
	}

	// get router
	r, ok := options.Values().Get("network.router")
	if ok {
		net.router = r.(router.Router)
	} else {
		net.router = router.DefaultRouter
	}

	// get proxy
	p, ok := options.Values().Get("network.proxy")
	if ok {
		net.proxy = p.(proxy.Proxy)
	} else {
		net.proxy = new(mucp.Proxy)
	}

	// get resolver
	res, ok := options.Values().Get("network.resolver")
	if ok {
		net.resolver = res.(resolver.Resolver)
	} else {
		net.resolver = new(registry.Resolver)
	}

	return net
}
