package model

// Options for model configuration
type Options struct {
	// Additional options can be added here
}

// Option is a function that modifies Options
type Option func(*Options)

func newOptions(opts ...Option) Options {
	options := Options{}
	for _, o := range opts {
		o(&options)
	}
	return options
}
