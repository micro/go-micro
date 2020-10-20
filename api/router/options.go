package router

import (
	"github.com/asim/go-micro/v3/api/resolver"
	"github.com/asim/go-micro/v3/api/resolver/path"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/v3/registry/memory"
)

type Options struct {
	Handler  string
	Registry registry.Registry
	Resolver resolver.Resolver
}

type Option func(o *Options)

func NewOptions(opts ...Option) Options {
	options := Options{
		Handler:  "meta",
		Registry: memory.NewRegistry(),
	}

	for _, o := range opts {
		o(&options)
	}

	if options.Resolver == nil {
		options.Resolver = path.NewResolver(
			resolver.WithHandler(options.Handler),
		)
	}

	return options
}

func WithHandler(h string) Option {
	return func(o *Options) {
		o.Handler = h
	}
}

func WithRegistry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

func WithResolver(r resolver.Resolver) Option {
	return func(o *Options) {
		o.Resolver = r
	}
}
