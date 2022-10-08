package selector

import (
	"context"

	"go-micro.dev/v4/logger"
	"go-micro.dev/v4/registry"
)

type Options struct {
	Registry registry.Registry
	Strategy Strategy

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
	// Logger is the underline logger
	Logger logger.Logger
}

type SelectOptions struct {
	Filters  []Filter
	Strategy Strategy

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type Option func(*Options)

// SelectOption used when making a select call.
type SelectOption func(*SelectOptions)

// Registry sets the registry used by the selector.
func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

// SetStrategy sets the default strategy for the selector.
func SetStrategy(fn Strategy) Option {
	return func(o *Options) {
		o.Strategy = fn
	}
}

// WithFilter adds a filter function to the list of filters
// used during the Select call.
func WithFilter(fn ...Filter) SelectOption {
	return func(o *SelectOptions) {
		o.Filters = append(o.Filters, fn...)
	}
}

// Strategy sets the selector strategy.
func WithStrategy(fn Strategy) SelectOption {
	return func(o *SelectOptions) {
		o.Strategy = fn
	}
}

// WithLogger sets the underline logger.
func WithLogger(l logger.Logger) Option {
	return func(o *Options) {
		o.Logger = l
	}
}
