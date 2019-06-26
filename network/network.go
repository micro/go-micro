// Package network is a package for defining a network overlay
package network

import (
	"github.com/micro/go-micro/config/options"
)

// Network is an interface defining a network
type Network interface {
	options.Options
	// Id of this node
	Id() string
	// Connect to the network
	Connect() (Node, error)
	// Peer with a neighboring network
	Peer(Network) (Link, error)
	// Retrieve list of connections
	Links() ([]Link, error)
}

// Node represents a single node on a network
type Node interface {
	// Node is a network. Network is a node.
	Network
	// Address of the node
	Address() string
	// Close the network connection
	Close() error
	// Accept messages on the network
	Accept() (*Message, error)
	// Send a message to the network
	Send(*Message) error
}

// Link is a connection between one network and another
type Link interface {
	// remote node the link is to
	Node
	// length of link which dictates speed
	Length() int
	// weight of link which dictates curvature
	Weight() int
}

// Message is the base type for opaque data
type Message struct {
	// Headers which provide local/remote info
	Header map[string]string
	// The opaque data being sent
	Data []byte
}

var (
	// TODO: set default network
	DefaultNetwork Network
)
