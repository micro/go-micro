// Package mucp initialises a mucp service
package mucp

import (
	// TODO: change to go-micro/service
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/client/mucp"
	"github.com/micro/go-micro/server/mucp"
)

// NewService returns a new mucp service
func NewService(opts ...micro.Option) micro.Service {
	options := []micro.Option{
		micro.Client(mucp.NewClient()),
		micro.Server(mucp.NewServer()),
	}

	options = append(options, opts...)

	return micro.NewService(opts...)
}
