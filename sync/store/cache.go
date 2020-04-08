// Package store syncs multiple go-micro stores
package store

import (
	"fmt"
	"sync"
	"time"

	"github.com/ef-ds/deque"
	"github.com/micro/go-micro/v2/store"
	"github.com/pkg/errors"
)

// Cache implements a cache in front of go-micro Stores
type Cache interface {
	store.Store

	// Force a full sync
	Sync() error
}
type cache struct {
	sOptions            store.Options
	cOptions            Options
	pendingWrites       []*deque.Deque
	pendingWriteTickers []*time.Ticker
	sync.RWMutex
}

// NewCache returns a new Cache
func NewCache(opts ...Option) Cache {
	c := &cache{}
	for _, o := range opts {
		o(&c.cOptions)
	}
	if c.cOptions.SyncInterval == 0 {
		c.cOptions.SyncInterval = 1 * time.Minute
	}
	if c.cOptions.SyncMultiplier == 0 {
		c.cOptions.SyncMultiplier = 5
	}
	return c
}

func (c *cache) Close() error {
	return nil
}

// Init initialises the storeOptions
func (c *cache) Init(opts ...store.Option) error {
	for _, o := range opts {
		o(&c.sOptions)
	}
	if len(c.cOptions.Stores) == 0 {
		return errors.New("the cache has no stores")
	}
	if c.sOptions.Context == nil {
		return errors.New("please provide a context to the cache. Cancelling the context signals that the cache is being disposed and syncs the cache")
	}
	for _, s := range c.cOptions.Stores {
		if err := s.Init(); err != nil {
			return errors.Wrapf(err, "Store %s failed to Init()", s.String())
		}
	}
	c.pendingWrites = make([]*deque.Deque, len(c.cOptions.Stores)-1)
	c.pendingWriteTickers = make([]*time.Ticker, len(c.cOptions.Stores)-1)
	for i := 0; i < len(c.pendingWrites); i++ {
		c.pendingWrites[i] = deque.New()
		c.pendingWrites[i].Init()
		c.pendingWriteTickers[i] = time.NewTicker(c.cOptions.SyncInterval * time.Duration(intpow(c.cOptions.SyncMultiplier, int64(i))))
	}
	go c.cacheManager()
	return nil
}

// Options returns the cache's store options
func (c *cache) Options() store.Options {
	return c.sOptions
}

// String returns a printable string describing the cache
func (c *cache) String() string {
	backends := make([]string, len(c.cOptions.Stores))
	for i, s := range c.cOptions.Stores {
		backends[i] = s.String()
	}
	return fmt.Sprintf("cache %v", backends)
}

func (c *cache) List(opts ...store.ListOption) ([]string, error) {
	return c.cOptions.Stores[0].List(opts...)
}

func (c *cache) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	return c.cOptions.Stores[0].Read(key, opts...)
}

func (c *cache) Write(r *store.Record, opts ...store.WriteOption) error {
	return c.cOptions.Stores[0].Write(r, opts...)
}

// Delete removes a key from the cache
func (c *cache) Delete(key string, opts ...store.DeleteOption) error {
	return c.cOptions.Stores[0].Delete(key, opts...)
}

func (c *cache) Sync() error {
	return nil
}

type internalRecord struct {
	key       string
	value     []byte
	expiresAt time.Time
}
