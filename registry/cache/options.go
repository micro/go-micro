package cache

import (
	"time"

	"go-micro.dev/v5/logger"
)

// WithTTL sets the cache TTL.
func WithTTL(t time.Duration) Option {
	return func(o *Options) {
		o.TTL = t
	}
}

// WithLogger sets the underline logger.
func WithLogger(l logger.Logger) Option {
	return func(o *Options) {
		o.Logger = l
	}
}

// WithMinimumRetryInterval sets the minimum retry interval for failed lookups.
// This prevents cache penetration when registry is failing and there's no stale cache.
func WithMinimumRetryInterval(d time.Duration) Option {
	return func(o *Options) {
		o.MinimumRetryInterval = d
	}
}
