// Package tunnel provides a network tunnel ontop of a link
package tunnel

import (
	"github.com/micro/go-micro/network/link"
	"github.com/micro/go-micro/transport"
)

// Tunnel creates a network tunnel on top of a link.
// It establishes multiple streams using the Micro-Tunnel header
// created as a hash of the address.
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

type Conn interface {
	// Specifies the tunnel id
	Id() string
	// a transport socket
	transport.Socket
}

// NewTunnel creates a new tunnel on top of a link
func NewTunnel(l link.Link) Tunnel {
	return newTunnel(l)
}
