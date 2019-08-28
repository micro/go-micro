// Package network is for creating internetworks
package network

import (
	"time"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/server"
)

var (
	// DefaultName is default network name
	DefaultName = "go.micro"
	// DefaultAddress is default network address
	DefaultAddress = ":0"
	// ResolveTime defines time interval to periodically resolve network nodes
	ResolveTime = 1 * time.Minute
	// AnnounceTime defines time interval to periodically announce node neighbours
	AnnounceTime = 30 * time.Second
)

// Network is micro network
type Network interface {
	// Name of the network
	Name() string
	// Address returns network bind address
	Address() string
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
