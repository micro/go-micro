// Package rpc provides a network agnostic RPC server
package rpc

import (
	"github.com/asim/nitro/app/server"
)

var (
	DefaultRouter = newRpcRouter()
)

// NewServer returns a micro server interface
func NewServer(opts ...server.Option) server.Server {
	return newServer(opts...)
}
