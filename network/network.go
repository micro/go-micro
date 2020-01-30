// Package network is for creating internetworks
package network

import (
	"time"

	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/server"
)

var (
	// DefaultName is default network name
	DefaultName = "go.micro"
	// DefaultAddress is default network address
	DefaultAddress = ":0"
	// ResolveTime defines time interval to periodically resolve network nodes
	ResolveTime = 1 * time.Minute
	// AnnounceTime defines time interval to periodically announce node neighbours
	AnnounceTime = 1 * time.Second
	// KeepAliveTime is the time in which we want to have sent a message to a peer
	KeepAliveTime = 30 * time.Second
	// SyncTime is the time a network node requests full sync from the network
	SyncTime = 1 * time.Minute
	// PruneTime defines time interval to periodically check nodes that need to be pruned
	// due to their not announcing their presence within this time interval
	PruneTime = 90 * time.Second
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

// NewNetwork returns a new network interface
func NewNetwork(opts ...Option) Network {
	return newNetwork(opts...)
}
