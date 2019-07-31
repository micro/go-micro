// Package network is for building peer to peer networks
package network

// Network is a
type Network interface {
	// Name of the network
	Name() string
	// Connect starts the network node
	Connect() error
	// Close shutsdown the network node
	Close() error
}

var (
	DefaultName = "go.micro.network"

	DefaultNetwork = NewNetwork()
)
