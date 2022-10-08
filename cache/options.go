package cache

import (
	"context"
	"time"

	"go-micro.dev/v4/logger"
)

// Options represents the options for the cache.
type Options struct {
	Expiration time.Duration
	Items      map[string]Item
	// Address represents the address or other connection information of the cache service.
	Address string
	// Context should contain all implementation specific options, using context.WithValue.
	Context context.Context
	// Logger is the be used logger
	Logger logger.Logger
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

// WithAddress sets the cache service address or connection information.
func WithAddress(addr string) Option {
	return func(o *Options) {
		o.Address = addr
	}
}

// WithContext sets the cache context, for any extra configuration.
func WithContext(c context.Context) Option {
	return func(o *Options) {
		o.Context = c
	}
}

// WithLogger sets underline logger.
func WithLogger(l logger.Logger) Option {
	return func(o *Options) {
		o.Logger = l
	}
}

// NewOptions returns a new options struct.
func NewOptions(opts ...Option) Options {
	options := Options{
		Expiration: DefaultExpiration,
		Items:      make(map[string]Item),
		Logger:     logger.DefaultLogger,
	}

	for _, o := range opts {
		o(&options)
	}

	return options
}
