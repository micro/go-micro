package cache

import "time"

// Options represents the options for the cache.
type Options struct {
	Expiration time.Duration
}

// Option manipulates the Options passed.
type Option func(o *Options)

// Expiration sets the duration for items stored in the cache to expire.
func Expiration(d time.Duration) Option {
	return func(o *Options) {
		o.Expiration = d
	}
}

// NewOptions returns a new options struct.
func NewOptions(opts ...Option) Options {
	options := Options{
		Expiration: DefaultExpiration,
	}

	for _, o := range opts {
		o(&options)
	}

	return options
}
