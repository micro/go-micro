// Package network is a package for defining a network overlay
package network

import (
	"github.com/micro/go-micro/config/options"
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
	return newNetwork(opts...)
}
