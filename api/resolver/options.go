package resolver

import (
	"net/http"
)

// NewOptions returns new initialised options
func NewOptions(opts ...Option) Options {
	var options Options
	for _, o := range opts {
		o(&options)
	}

	if options.Namespace == nil {
		options.Namespace = StaticNamespace("go.micro")
	}

	return options
}

// WithHandler sets the handler being used
func WithHandler(h string) Option {
	return func(o *Options) {
		o.Handler = h
	}
}

// WithNamespace sets the function which determines the namespace for a request
func WithNamespace(n func(*http.Request) string) Option {
	return func(o *Options) {
		o.Namespace = n
	}
}
