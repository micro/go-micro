package resolver

import (
	"github.com/micro/go-micro/v2/registry"
)

type Options struct {
	Handler       string
	Domain     string
	ServicePrefix string
}

type Option func(o *Options)

// WithHandler sets the handler being used
func WithHandler(h string) Option {
	return func(o *Options) {
		o.Handler = h
	}
}

// WithDomain sets the namespace option
func WithDomain(n string) Option {
	return func(o *Options) {
		o.Domain = n
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
	if len(options.Domain) == 0 {
		options.Domain = registry.DefaultDomain
	}

	return options
}
