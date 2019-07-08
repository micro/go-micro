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
	// Create starts the network
	Create() (*Node, error)
	// Name of the network
	Name() string
	// Connect to a node
	Connect(*Node) (Conn, error)
	// Listen for connections
	Listen(*Node) (Listener, error)
}

type Node struct {
	Id       string
	Address  string
	Metadata map[string]string
}

type Listener interface {
	Address() string
	Close() error
	Accept() (Conn, error)
}

type Conn interface {
	// Unique id of the connection
	Id() string
	// Close the connection
	Close() error
	// Send a message
	Send(*Message) error
	// Receive a message
	Recv(*Message) error
	// The remote node
	Remote() string
	// The local node
	Local() string
}

type Message struct {
	Header map[string]string
	Body   []byte
}

var (
	// The default network name is local
	DefaultName = "go.micro"

	// just the standard network element
	DefaultNetwork = NewNetwork()
)

// NewNetwork returns a new network interface
func NewNetwork(opts ...options.Option) Network {
	return newNetwork(opts...)
}
