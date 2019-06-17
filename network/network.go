// Package network is an interface for defining a network overlay
package network

import (
	"github.com/micro/go-micro/config/options"
)

type Network interface {
	options.Options
	// Id of this network
	Id() string
	// Connect to the network with id
	Connect(id string) error
	// Close the network connection
	Close() error
	// Accept messages
	Accept() (*Message, error)
	// Send a message
	Send(*Message) error
	// Advertise a service on this network
	Advertise(service string) error
	// Retrieve list of nodes for a service
	Nodes(service string) ([]Node, error)
}

// Node represents a network node
type Node interface {
	// Id of the node
	Id() string
	// The network for this node
	Network() Network
}

// Message is a message sent over the network
type Message struct {
	// Headers are the routing headers
	// e.g Micro-Service, Micro-Endpoint, Micro-Network
	// see https://github.com/micro/development/blob/master/protocol.md
	Header map[string]string
	// Body is the encaspulated payload
	Body []byte
}
