package api

import (
	"go-micro.dev/v4/api/router"
	registry2 "go-micro.dev/v4/api/router/registry"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/registry"
)

func NewOptions(opts ...Option) Options {
	options := Options{
		Address: ":8080",
	}

	for _, o := range opts {
		o(&options)
	}

	return options
}

// WithRouter sets the router to use e.g static or registry.
func WithRouter(r router.Router) Option {
	return func(o *Options) error {
		o.Router = r
		return nil
	}
}

// WithRegistry sets the api's client and router to use registry.
func WithRegistry(r registry.Registry) Option {
	return func(o *Options) error {
		o.Client = client.NewClient(client.Registry(r))
		o.Router = registry2.NewRouter(router.WithRegistry(r))
		return nil
	}
}
