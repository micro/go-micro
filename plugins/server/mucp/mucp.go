// Package mucp provides an mucp server
package mucp

import (
	"github.com/chinahtl/go-micro/v3/cmd"
	"github.com/chinahtl/go-micro/v3/server"
)

func init() {
	cmd.DefaultServers["mucp"] = NewServer
}

// NewServer returns a micro server interface
func NewServer(opts ...server.Option) server.Server {
	return server.NewServer(opts...)
}
