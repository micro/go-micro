package cache

import (
	"time"

	"github.com/micro/go-micro/v2/store"
)

// Options represents Cache options
type Options struct {
	// Stores represents layers in the cache in ascending order. L0, L1, L2, etc
	Stores []store.Store
	// SyncInterval is the duration between syncs from L0 to L1
	SyncInterval time.Duration
	// SyncMultiplier is the multiplication factor between each store.
	SyncMultiplier int64
}

// Option sets Cache Options
type Option func(o *Options)

// Stores sets the layers that make up the cache
func Stores(stores ...store.Store) Option {
	return func(o *Options) {
		o.Stores = make([]store.Store, len(stores))
		for i, s := range stores {
			o.Stores[i] = s
		}
	}
}

// SyncInterval sets the duration between syncs from L0 to L1
func SyncInterval(d time.Duration) Option {
	return func(o *Options) {
		o.SyncInterval = d
	}
}

// SyncMultiplier sets the multiplication factor for time to wait each cache layer
func SyncMultiplier(i int64) Option {
	return func(o *Options) {
		o.SyncMultiplier = i
	}
}
