package network

import (
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/server"
)

type Option func(*Options)

type Options struct {
	Name   string
	Client client.Client
	Server server.Server
}

// The network name
func Name(n string) Option {
	return func(o *Options) {
		o.Name = n
	}
}

// The network client
func Client(c client.Client) Option {
	return func(o *Options) {
		o.Client = c
	}
}

// The network server
func Server(s server.Server) Option {
	return func(o *Options) {
		o.Server = s
	}
}
