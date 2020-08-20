package store

import (
	"time"

	"github.com/micro/go-micro/v3/store"
)

type Options struct {
	Store store.Store
	TTL   time.Duration
}

type Option func(o *Options)

// WithStore sets the underlying store to use
func WithStore(s store.Store) Option {
	return func(o *Options) {
		o.Store = s
	}
}

// WithTTL sets the default TTL
func WithTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.TTL = ttl
	}
}
