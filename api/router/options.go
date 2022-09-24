package router

import (
	"go-micro.dev/v4/api/resolver"
	"go-micro.dev/v4/api/resolver/vpath"
	"go-micro.dev/v4/logger"
	"go-micro.dev/v4/registry"
)

type Options struct {
	Handler  string
	Registry registry.Registry
	Resolver resolver.Resolver
	Logger   *logger.Helper
}

type Option func(o *Options)

func NewOptions(opts ...Option) Options {
	options := Options{
		Handler:  "meta",
		Registry: registry.DefaultRegistry,
		Logger:   logger.DefaultHelper,
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

// WithLogger sets the underline logging framework
func WithLogger(l logger.Logger) Option {
	return func(o *Options) {
		o.Logger = logger.NewHelper(l)
	}
}
