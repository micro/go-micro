package cache

import (
	"github.com/micro/go-micro/v3/store"
	"github.com/micro/go-micro/v3/store/memory"
)

// cache store is a store with caching to reduce IO where applicable.
// A memory store is used to cache reads from the given backing store.
// Reads are read through, writes are write-through
type cache struct {
	m       store.Store // the memory store
	b       store.Store // the backing store, could be file, cockroach etc
	options store.Options
}

// NewStore returns a new cache store
func NewStore(store store.Store, opts ...store.Option) store.Store {
	cf := &cache{
		m: memory.NewStore(opts...),
		b: store,
	}
	return cf

}

func (c *cache) init(opts ...store.Option) error {
	for _, o := range opts {
		o(&c.options)
	}
	return nil
}

// Init initialises the underlying stores
func (c *cache) Init(opts ...store.Option) error {
	if err := c.init(opts...); err != nil {
		return err
	}
	if err := c.m.Init(opts...); err != nil {
		return err
	}
	return c.b.Init(opts...)
}

// Options allows you to view the current options.
func (c *cache) Options() store.Options {
	return c.options
}

// Read takes a single key name and optional ReadOptions. It returns matching []*Record or an error.
func (c *cache) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	recs, err := c.m.Read(key, opts...)
	if err != nil && err != store.ErrNotFound {
		return nil, err
	}
	if len(recs) > 0 {
		return recs, nil
	}
	recs, err = c.b.Read(key, opts...)
	if err == nil {
		for _, rec := range recs {
			if err := c.m.Write(rec); err != nil {
				return nil, err
			}
		}
	}
	return recs, err
}

// Write() writes a record to the store, and returns an error if the record was not written.
// If the write succeeds in writing to memory but fails to write through to file, you'll receive an error
// but the value may still reside in memory so appropriate action should be taken.
func (c *cache) Write(r *store.Record, opts ...store.WriteOption) error {
	if err := c.m.Write(r, opts...); err != nil {
		return err
	}
	return c.b.Write(r, opts...)
}

// Delete removes the record with the corresponding key from the store.
// If the delete succeeds in writing to memory but fails to write through to file, you'll receive an error
// but the value may still reside in memory so appropriate action should be taken.
func (c *cache) Delete(key string, opts ...store.DeleteOption) error {
	if err := c.m.Delete(key, opts...); err != nil {
		return err
	}
	return c.b.Delete(key, opts...)
}

// List returns any keys that match, or an empty list with no error if none matched.
func (c *cache) List(opts ...store.ListOption) ([]string, error) {
	keys, err := c.m.List(opts...)
	if err != nil && err != store.ErrNotFound {
		return nil, err
	}
	if len(keys) > 0 {
		return keys, nil
	}
	keys, err = c.b.List(opts...)
	if err == nil {
		for _, key := range keys {
			recs, err := c.b.Read(key)
			if err != nil {
				return nil, err
			}
			for _, r := range recs {
				if err := c.m.Write(r); err != nil {
					return nil, err
				}
			}

		}
	}
	return keys, err
}

// Close the store and the underlying store
func (c *cache) Close() error {
	if err := c.m.Close(); err != nil {
		return err
	}
	return c.b.Close()
}

// String returns the name of the implementation.
func (c *cache) String() string {
	return "cache"
}
