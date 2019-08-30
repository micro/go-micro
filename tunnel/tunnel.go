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
	// Address the tunnel is listening on
	Address() string
	// Connect connects the tunnel
	Connect() error
	// Close closes the tunnel
	Close() error
	// Connect to a channel
	Dial(channel string) (Session, error)
	// Accept connections on a channel
	Listen(channel string) (Listener, error)
	// Name of the tunnel implementation
	String() string
}

// The listener provides similar constructs to the transport.Listener
type Listener interface {
	Channel() string
	Close() error
	Accept() (Session, error)
}

// Session is a unique session created when dialling or accepting connections on the tunnel
type Session interface {
	// Specifies the tunnel id
	Id() string
	// The session
	Channel() string
	// a transport socket
	transport.Socket
}

// NewTunnel creates a new tunnel
func NewTunnel(opts ...Option) Tunnel {
	return newTunnel(opts...)
}
