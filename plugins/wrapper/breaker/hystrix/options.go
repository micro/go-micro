package hystrix

import "context"

// Options represents hystrix client wrapper options
type Options struct {
	Filter   func(context.Context, error) error
	Fallback func(context.Context, error) error
}

// Option represents options update func
type Option func(*Options)

// WithFilter used to set filter func for options
func WithFilter(filter func(context.Context, error) error) Option {
	return func(o *Options) {
		o.Filter = filter
	}
}

// WithFallback used to set fallback func for options
func WithFallback(fallback func(context.Context, error) error) Option {
	return func(o *Options) {
		o.Fallback = fallback
	}
}
