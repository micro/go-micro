package router

import (
	"go-micro.dev/v4/api/resolver"
	"go-micro.dev/v4/api/resolver/vpath"
	"go-micro.dev/v4/logger"
	"go-micro.dev/v4/registry"
)

// Options is a struct of options available.
type Options struct {
	Handler  string
	Registry registry.Registry
	Resolver resolver.Resolver
	Logger   logger.Logger
}

// Option is a helper for a single options.
type Option func(o *Options)

// NewOptions wires options together.
func NewOptions(opts ...Option) Options {
	options := Options{
		Handler:  "meta",
		Registry: registry.DefaultRegistry,
		Logger:   logger.DefaultLogger,
	}

	for _, o := range opts {
		o(&options)
	}

	if options.Resolver == nil {
		options.Resolver = vpath.NewResolver(
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

// WithLogger sets the underline logger.
func WithLogger(l logger.Logger) Option {
	return func(o *Options) {
		o.Logger = l
	}
}
