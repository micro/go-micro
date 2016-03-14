package rpc

import (
	"github.com/micro/go-micro/server"
)

func NewServer(opts ...server.Option) server.Server {
	return server.NewServer(opts...)
}
