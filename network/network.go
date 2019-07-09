// Package network is a package for defining a network overlay
package network

import (
	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/network/transport"
)

// Network defines a network interface. The network is a single
// shared network between all nodes connected to it. The network
// is responsible for routing messages to the correct services.
type Network interface {
	options.Options
	// Name of the network
	Name() string
	// Create returns a new network node id/address
	Create() (*Node, error)
	// Connect to a node on the network
	Connect(*Node) (Conn, error)
	// Listen for connections for this node
	Listen(*Node) (Listener, error)
}

// Node is a network node represented with id/address and
// metadata which includes the network name, transport, etc
type Node struct {
	Id       string
	Address  string
	Network  string
	Metadata map[string]string
}

// A network node listener which can be used to receive messages
type Listener interface {
	Address() string
	Close() error
	Accept() (Conn, error)
}

// A connection from another node on the network
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

// The message type sent over the network
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

// Name sets the network name
func Name(n string) options.Option {
	return options.WithValue("network.name", n)
}

// Address sets the network address
func Address(a string) options.Option {
	return options.WithValue("network.address", a)
}

// Transport sets the network transport
func Transport(t transport.Transport) options.Option {
	return options.WithValue("network.transport", t)
}
