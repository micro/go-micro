package resolver

import (
	"github.com/micro/go-micro/v3/registry"
)

type Options struct {
	Handler       string
	ServicePrefix string
}

type Option func(o *Options)

// WithHandler sets the handler being used
func WithHandler(h string) Option {
	return func(o *Options) {
		o.Handler = h
	}
}

// WithServicePrefix sets the ServicePrefix option
func WithServicePrefix(p string) Option {
	return func(o *Options) {
		o.ServicePrefix = p
	}
}

// NewOptions returns new initialised options
func NewOptions(opts ...Option) Options {
	var options Options
	for _, o := range opts {
		o(&options)
	}
	return options
}

// ResolveOptions are used when resolving a request
type ResolveOptions struct {
	Domain string
}

// ResolveOption sets an option
type ResolveOption func(*ResolveOptions)

// Domain sets the resolve Domain option
func Domain(n string) ResolveOption {
	return func(o *ResolveOptions) {
		o.Domain = n
	}
}

// NewResolveOptions returns new initialised resolve options
func NewResolveOptions(opts ...ResolveOption) ResolveOptions {
	var options ResolveOptions
	for _, o := range opts {
		o(&options)
	}
	if len(options.Domain) == 0 {
		options.Domain = registry.DefaultDomain
	}

	return options
}
