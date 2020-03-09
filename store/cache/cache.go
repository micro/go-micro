package cache

import (
	"github.com/micro/go-micro/v2/store"
	"github.com/pkg/errors"
)

// Cache implements a cache in front of a micro Store
type Cache struct {
	options store.Options
	store.Store

	stores []store.Store
}

// NewStore returns new cache
func NewStore(opts ...store.Option) store.Store {
	s := &Cache{
		options: store.Options{},
		stores:  []store.Store{},
	}
	for _, o := range opts {
		o(&s.options)
	}
	return s
}

// Init initialises a new cache
func (c *Cache) Init(opts ...store.Option) error {
	for _, o := range opts {
		o(&c.options)
	}
	for _, s := range c.stores {
		if err := s.Init(); err != nil {
			return errors.Wrapf(err, "Store %s failed to Init()", s.String())
		}
	}
	return nil
}
