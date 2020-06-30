package selector

import (
	"context"

	"github.com/micro/go-micro/v2/registry"
)

type Options struct {
	Registry registry.Registry
	Strategy Strategy

	// Domain to lookup services from within the registry
	Domain string

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type SelectOptions struct {
	Filters  []Filter
	Strategy Strategy
	Domain   string

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

// Option used to initialise the selector
type Option func(*Options)

// SelectOption used when making a select call
type SelectOption func(*SelectOptions)

// Registry sets the registry used by the selector
func Registry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

// Domain sets the domain used by the selector
func Domain(d string) Option {
	return func(o *Options) {
		o.Domain = d
	}
}

// SetStrategy sets the default strategy for the selector
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

// Strategy sets the selector strategy
func WithStrategy(fn Strategy) SelectOption {
	return func(o *SelectOptions) {
		o.Strategy = fn
	}
}

// WithDomain sets the registry domain to use for the selection
func WithDomain(d string) SelectOption {
	return func(o *SelectOptions) {
		o.Domain = d
	}
}
