package router

import (
	"github.com/micro/go-micro/api/resolver"
	"github.com/micro/go-micro/api/resolver/micro"
	"github.com/micro/go-micro/config/cmd"
	"github.com/micro/go-micro/registry"
)

type Options struct {
	Namespace string
	Handler   string
	Registry  registry.Registry
	Resolver  resolver.Resolver
}

type Option func(o *Options)

func NewOptions(opts ...Option) Options {
	options := Options{
		Handler:  "meta",
		Registry: *cmd.DefaultOptions().Registry,
	}

	for _, o := range opts {
		o(&options)
	}

	if options.Resolver == nil {
		options.Resolver = micro.NewResolver(
			resolver.WithHandler(options.Handler),
			resolver.WithNamespace(options.Namespace),
		)
	}

	return options
}

func WithHandler(h string) Option {
	return func(o *Options) {
		o.Handler = h
	}
}

func WithNamespace(ns string) Option {
	return func(o *Options) {
		o.Namespace = ns
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
