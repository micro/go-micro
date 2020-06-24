package cachedfile

import (
	"github.com/micro/go-micro/v2/store"
	"github.com/micro/go-micro/v2/store/file"
	"github.com/micro/go-micro/v2/store/memory"
)

// cachedFile store is a file store with caching to reduce IO where applicable.
// Reads are read through, writes are write-through
type cachedFile struct {
	m       store.Store // the memory store
	f       store.Store // the file store
	options store.Options
}

// NewStore returns a new cachedFile store
func NewStore(opts ...store.Option) store.Store {
	cf := &cachedFile{
		m: memory.NewStore(opts...),
		f: file.NewStore(opts...),
	}
	return cf

}

func (c *cachedFile) init(opts ...store.Option) error {
	for _, o := range opts {
		o(&c.options)
	}
	return nil
}

// Init initialises the store.
func (c *cachedFile) Init(opts ...store.Option) error {
	if err := c.init(opts...); err != nil {
		return err
	}
	if err := c.m.Init(); err != nil {
		return err
	}
	return c.f.Init()
}

// Options allows you to view the current options.
func (c *cachedFile) Options() store.Options {
	return c.options
}

// Read takes a single key name and optional ReadOptions. It returns matching []*Record or an error.
func (c *cachedFile) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	recs, err := c.m.Read(key, opts...)
	if err != nil && err != store.ErrNotFound {
		return nil, err
	}
	if len(recs) > 0 {
		return recs, nil
	}
	recs, err = c.f.Read(key, opts...)
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
func (c *cachedFile) Write(r *store.Record, opts ...store.WriteOption) error {
	if err := c.m.Write(r, opts...); err != nil {
		return err
	}
	return c.f.Write(r, opts...)
}

// Delete removes the record with the corresponding key from the store.
// If the delete succeeds in writing to memory but fails to write through to file, you'll receive an error
// but the value may still reside in memory so appropriate action should be taken.
func (c *cachedFile) Delete(key string, opts ...store.DeleteOption) error {
	if err := c.m.Delete(key, opts...); err != nil {
		return err
	}
	return c.f.Delete(key, opts...)
}

// List returns any keys that match, or an empty list with no error if none matched.
func (c *cachedFile) List(opts ...store.ListOption) ([]string, error) {
	keys, err := c.m.List(opts...)
	if err != nil && err != store.ErrNotFound {
		return nil, err
	}
	if len(keys) > 0 {
		return keys, nil
	}
	keys, err = c.f.List(opts...)
	if err == nil {
		for _, key := range keys {
			recs, err := c.f.Read(key)
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

// Close the store
func (c *cachedFile) Close() error {
	if err := c.m.Close(); err != nil {
		return err
	}
	return c.f.Close()
}

// String returns the name of the implementation.
func (c *cachedFile) String() string {
	return "cachedFile"
}
