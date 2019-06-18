// Package network is a package for defining a network overlay
package network

import (
	"github.com/micro/go-micro/config/options"
)

// Network is an interface defining networks or graphs
type Network interface {
	options.Options
	// Id of this node
	Id() uint64
	// Connect to a node
	Connect(id uint64) (Link, error)
	// Close the network connection
	Close() error
	// Accept messages on the network
	Accept() (*Message, error)
	// Send a message to the network
	Send(*Message) error
	// Retrieve list of connections
	Links() ([]Link, error)
}

// Node represents a network node
type Node interface {
	// Node is a network. Network is a node.
	Network
}

// Link is a connection to another node
type Link interface {
	// remote node
	Node
	// length of link which dictates speed
	Length() int
	// weight of link which dictates curvature
	Weight() int
}

// Message is the base type for opaque data
type Message []byte

var (
	// TODO: set default network
	DefaultNetwork Network
)
