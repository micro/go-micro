// Package mucp provides a transport agnostic RPC server
package mucp

import (
	"github.com/asim/nitro/v3/server"
)

var (
	DefaultRouter = newRpcRouter()
)

// NewServer returns a micro server interface
func NewServer(opts ...server.Option) server.Server {
	return newServer(opts...)
}
