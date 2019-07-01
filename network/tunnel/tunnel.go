// Package tunnel provides a network tunnel
package tunnel

import (
	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/transport"
)

// Tunnel creates a network tunnel
type Tunnel interface {
	// Connect connects the tunnel
	Connect() error
	// Close closes the tunnel
	Close() error
	// Dial an endpoint
	Dial(addr string) (Conn, error)
	// Accept connections
	Accept(addr string) (Conn, error)
}

// Conn return a transport socket with a unique id.
// This means Conn can be used as a transport.Socket
type Conn interface {
	// Unique id of the connection
	Id() string
	// Underlying socket
	transport.Socket
}

// A network interface to use for sending/receiving.
// When Tunnel.Connect is called it starts processing
// messages over the interface.
type Interface interface {
	// Address of the interface
	Addr() string
	// Receive new messages
	Recv() (*Message, error)
	// Send messages
	Send(*Message) error
}

// Messages received over the interface
type Message struct {
	Header map[string]string
	Body   []byte
}

// NewTunnel creates a new tunnel
func NewTunnel(opts ...options.Option) Tunnel {
	options := options.NewOptions(opts...)

	i, ok := options.Values().Get("tunnel.net")
	if !ok {
		// wtf
		return nil
	}

	return newTunnel(i.(Interface))
}

// WithInterface passes in the interface
func WithInterface(net Interface) options.Option {
	return options.WithValue("tunnel.net", net)
}
