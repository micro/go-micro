package cache

import "time"

// Options represents the options for the cache.
type Options struct {
	Expiration time.Duration
	Items      map[string]Item
}

// Option manipulates the Options passed.
type Option func(o *Options)

// Expiration sets the duration for items stored in the cache to expire.
func Expiration(d time.Duration) Option {
	return func(o *Options) {
		o.Expiration = d
	}
}

// Items initializes the cache with preconfigured items.
func Items(i map[string]Item) Option {
	return func(o *Options) {
		o.Items = i
	}
}

// NewOptions returns a new options struct.
func NewOptions(opts ...Option) Options {
	options := Options{
		Expiration: DefaultExpiration,
		Items:      make(map[string]Item),
	}

	for _, o := range opts {
		o(&options)
	}

	return options
}
