// Package mucp provides an mucp server
package mucp

import (
	"github.com/micro/go-micro/v3/server"
)

var (
	DefaultRouter = newRpcRouter()
)

// NewServer returns a micro server interface
func NewServer(opts ...server.Option) server.Server {
	return newServer(opts...)
}
