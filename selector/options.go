package selector

import "github.com/micro/go-micro/v3/router"

// Options used to configure a selector
type Options struct{}

// Option updates the options
type Option func(*Options)

// Filter the routes
type Filter func([]router.Route) []router.Route

// SelectOptions used to configure selection
type SelectOptions struct {
	Filters []Filter
}

// SelectOption updates the select options
type SelectOption func(*SelectOptions)

// WithFilter adds a filter to the options
func WithFilter(f Filter) SelectOption {
	return func(o *SelectOptions) {
		o.Filters = append(o.Filters, f)
	}
}

// NewSelectOptions parses select options
func NewSelectOptions(opts ...SelectOption) SelectOptions {
	var options SelectOptions
	for _, o := range opts {
		o(&options)
	}

	if options.Filters == nil {
		options.Filters = make([]Filter, 0)
	}

	return options
}
