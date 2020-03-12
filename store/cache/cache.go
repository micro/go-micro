package cache

import (
	"fmt"

	"github.com/micro/go-micro/v2/store"
	"github.com/pkg/errors"
)

// Cache implements a cache in front of a micro Store
type Cache interface {
	store.Store

	SyncNow() error
}
type cache struct {
	stores  []store.Store
	options store.Options
}

// NewCache returns a new Cache
func NewCache(stores ...store.Store) Cache {
	c := &cache{
		stores: stores,
	}
	return c
}

// Init initialises a new cache
func (c *cache) Init(opts ...store.Option) error {
	for _, o := range opts {
		o(&c.options)
	}
	if c.options.Context == nil {
		return errors.New("please provide a context to the cache. Cancelling the context signals that the cache is being disposed and syncs the cache")
	}
	for _, s := range c.stores {
		if err := s.Init(); err != nil {
			return errors.Wrapf(err, "Store %s failed to Init()", s.String())
		}
	}
	return nil
}

// Options returns the cache's store options
func (c *cache) Options() store.Options {
	return c.options
}

// String returns a printable string describing the cache
func (c *cache) String() string {
	backends := make([]string, len(c.stores))
	for i, s := range c.stores {
		backends[i] = s.String()
	}
	return fmt.Sprintf("cache %v", backends)
}

func (c *cache) List(opts ...store.ListOption) ([]string, error) {
	return c.stores[0].List(opts...)
}

func (c *cache) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	return c.stores[0].Read(key, opts...)
}

func (c *cache) Write(r *store.Record, opts ...store.WriteOption) error {
	return c.stores[0].Write(r, opts...)
}

// Delete removes a key from the cache
func (c *cache) Delete(key string, opts ...store.DeleteOption) error {
	return c.stores[0].Delete(key, opts...)
}

func (c *cache) SyncNow() error {
	return nil
}
