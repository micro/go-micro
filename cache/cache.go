package cache

import (
	"context"
	"errors"
	"time"
)

var (
	// DefaultCache is the default cache.
	DefaultCache Cache = NewCache()
	// DefaultExpiration is the default duration for items stored in
	// the cache to expire.
	DefaultExpiration time.Duration = 0

	// ErrItemExpired is returned in Cache.Get when the item found in the cache
	// has expired.
	ErrItemExpired error = errors.New("item has expired")
	// ErrKeyNotFound is returned in Cache.Get and Cache.Delete when the
	// provided key could not be found in cache.
	ErrKeyNotFound error = errors.New("key not found in cache")
)

// Cache is the interface that wraps the cache.
//
// Context specifies the context for the cache.
// Get gets a cached value by key.
// Put stores a key-value pair into cache.
// Delete removes a key from cache.
type Cache interface {
	Context(ctx context.Context) Cache
	Get(key string) (interface{}, time.Time, error)
	Put(key string, val interface{}, d time.Duration) error
	Delete(key string) error
}

// Item represents an item stored in the cache.
type Item struct {
	Value      interface{}
	Expiration int64
}

// Expired returns true if the item has expired.
func (i *Item) Expired() bool {
	if i.Expiration == 0 {
		return false
	}

	return time.Now().UnixNano() > i.Expiration
}

// NewCache returns a new cache.
func NewCache(opts ...Option) Cache {
	options := NewOptions(opts...)
	items := make(map[string]Item)

	if len(options.Items) > 0 {
		items = options.Items
	}

	return &memCache{
		opts:  options,
		items: items,
	}
}
