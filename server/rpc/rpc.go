// Package rpc provides a transport agnostic RPC server
package rpc

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
