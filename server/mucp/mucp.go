// Package mucp provides an mucp server
package mucp

import (
	"github.com/micro/go-micro/v2/server"
)

// NewServer returns a micro server interface
func NewServer(opts ...server.Option) server.Server {
	return server.NewServer(opts...)
}
