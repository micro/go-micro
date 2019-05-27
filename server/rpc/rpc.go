// Package rpc provides an rpc server
package rpc

import (
	"github.com/micro/go-micro/server"
)

// NewServer returns a micro server interface
func NewServer(opts ...server.Option) server.Server {
	return server.NewServer(opts...)
}
