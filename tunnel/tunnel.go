// Package tunnel provides micro network tunnelling
package tunnel

import (
	"github.com/micro/go-micro/transport"
)

// Tunnel creates a p2p network tunnel.
type Tunnel interface {
	transport.Transport
	// Connect connects the tunnel
	Connect() error
	// Close closes the tunnel
	Close() error
}
