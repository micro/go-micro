// Network is a package for peer-to-peer networking
package network

// Network represents a p2p network
type Network interface {
	// Connect to the network
	Connect() error
	// Close the network connection
	Close() error
}
