// Package tunnel provides micro network tunnelling
package tunnel

import (
	"github.com/micro/go-micro/transport"
)

// Status is tunnel status
type Status int

const (
	// Connected means the tunnel is alive
	Connected Status = iota
	// Closed meands the tunnel has been disconnected
	Closed
)

// Tunnel creates a p2p network tunnel.
type Tunnel interface {
	// Id returns tunnel id
	Id() string
	// Address returns tunnel address
	Address() string
	// Tramsport returns tunnel transport
	Transport() transport.Transport
	// Connect connects the tunnel
	Connect() error
	// Close closes the tunnel
	Close() error
	// Status returns tunnel status
	Status() Status
}

// NewTunnel creates a new tunnel on top of a link
func NewTunnel(opts ...Option) Tunnel {
	return newTunnel(opts...)
}
