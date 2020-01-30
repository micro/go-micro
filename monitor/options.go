package monitor

import (
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/registry"
)

type Options struct {
	Client   client.Client
	Registry registry.Registry
}

type Option func(*Options)

func Client(c client.Client) Option {
	return func(o *Options) {
		o.Client = c
	}
}

func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}
