package selector

// Options used to configure a selector
type Options struct{}

// Option updates the options
type Option func(*Options)

// SelectOptions used to configure selection
type SelectOptions struct{}

// SelectOption updates the select options
type SelectOption func(*SelectOptions)

// NewSelectOptions parses select options
func NewSelectOptions(opts ...SelectOption) SelectOptions {
	var options SelectOptions
	for _, o := range opts {
		o(&options)
	}

	return options
}
