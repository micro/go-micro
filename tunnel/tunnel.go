// Package tunnel provides gre network tunnelling
package tunnel

import (
	"github.com/micro/go-micro/transport"
)

// Tunnel creates a gre network tunnel on top of a link.
// It establishes multiple streams using the Micro-Tunnel-Id header
// and Micro-Tunnel-Session header. The tunnel id is a hash of
// the address being requested.
type Tunnel interface {
	Init(opts ...Option) error
	// Connect connects the tunnel
	Connect() error
	// Close closes the tunnel
	Close() error
	// Dial an endpoint
	Dial(addr string) (Conn, error)
	// Accept connections
	Listen(addr string) (Listener, error)
}

// The listener provides similar constructs to the transport.Listener
type Listener interface {
	Addr() string
	Close() error
	Accept() (Conn, error)
}

// Conn is a connection dialed or accepted which includes the tunnel id and session
type Conn interface {
	// Specifies the tunnel id
	Id() string
	// The session
	Session() string
	// a transport socket
	transport.Socket
}

// NewTunnel creates a new tunnel
func NewTunnel(opts ...Option) Tunnel {
	return newTunnel(opts...)
}
