package selector

import (
	"github.com/micro/go-micro/registry"
)

type Options struct {
	Registry registry.Registry

	// Other options to be used by broker implementations
	Options map[string]string
}

type SelectOptions struct {
	Filters []SelectFilter

	// Other options to be used by broker implementations
	Options map[string]string
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

// Filter adds a filter function to the list of filters
// used during the Select call.
func Filter(fn SelectFilter) SelectOption {
	return func(o *SelectOptions) {
		o.Filters = append(o.Filters, fn)
	}
}
