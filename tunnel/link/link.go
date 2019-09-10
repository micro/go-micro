// Package link provides a measured link on top of a transport.Socket
package link

import (
	"errors"

	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/transport"
)

var (
	// ErrLinkClosed is returned when attempting i/o operation on the closed link
	ErrLinkClosed = errors.New("link closed")
)

// Link is a layer on top of a transport socket with the
// buffering send and recv queues with the ability to
// measure the actual transport link and reconnect if
// an address is specified.
type Link interface {
	// provides the transport.Socket interface
	transport.Socket
	// Connect connects the link. It must be called first
	// if there's an expectation to create a new socket.
	Connect() error
	// Id of the link is "local" if not specified
	Id() string
	// Status of the link
	Status() string
	// Depth of the buffers
	Weight() int
	// Rate of the link
	Length() int
}

// NewLink creates a new link on top of a socket
func NewLink(opts ...options.Option) Link {
	return newLink(options.NewOptions(opts...))
}

// Sets the link id which otherwise defaults to "local"
func Id(id string) options.Option {
	return options.WithValue("link.id", id)
}

// The address to use for the link. Connect must be
// called for this to be used, its otherwise unused.
func Address(a string) options.Option {
	return options.WithValue("link.address", a)
}

// The transport to use for the link where we
// want to dial the connection first.
func Transport(t transport.Transport) options.Option {
	return options.WithValue("link.transport", t)
}

// Socket sets the socket to use instead of dialing.
func Socket(s transport.Socket) options.Option {
	return options.WithValue("link.socket", s)
}
