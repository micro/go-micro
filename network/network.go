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
	DefaultName    = "go.micro.network"
	DefaultAddress = ":0"
	DefaultNetwork = NewNetwork()
)

// NewNetwork returns a new network interface
func NewNetwork(opts ...Option) Network {
	return newNetwork(opts...)
}
