// Package network is for creating internetworks
package network

import (
	"github.com/micro/go-micro/v3/client"
	"github.com/micro/go-micro/v3/server"
)

// Error is network node errors
type Error interface {
	// Count is current count of errors
	Count() int
	// Msg is last error message
	Msg() string
}

// Status is node status
type Status interface {
	// Error reports error status
	Error() Error
}

// Node is network node
type Node interface {
	// Id is node id
	Id() string
	// Address is node bind address
	Address() string
	// Peers returns node peers
	Peers() []Node
	// Network is the network node is in
	Network() Network
	// Status returns node status
	Status() Status
}

// Network is micro network
type Network interface {
	// Node is network node
	Node
	// Initialise options
	Init(...Option) error
	// Options returns the network options
	Options() Options
	// Name of the network
	Name() string
	// Connect starts the resolver and tunnel server
	Connect() error
	// Close stops the tunnel and resolving
	Close() error
	// Client is micro client
	Client() client.Client
	// Server is micro server
	Server() server.Server
}
