package selector

import (
	"github.com/micro/go-micro/registry"

	"golang.org/x/net/context"
)

type Options struct {
	Registry registry.Registry

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type SelectOptions struct {
	Filters []SelectFilter

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

// Filter adds a filter function to the list of filters
// used during the Select call.
func Filter(fn SelectFilter) SelectOption {
	return func(o *SelectOptions) {
		o.Filters = append(o.Filters, fn)
	}
}
