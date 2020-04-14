// Package syncs will sync multiple stores
package sync

import (
	"fmt"
	"sync"
	"time"

	"github.com/ef-ds/deque"
	"github.com/micro/go-micro/v2/store"
	"github.com/pkg/errors"
)

// Sync implements a sync in for stores
type Sync interface {
	// Implements the store interface
	store.Store
	// Force a full sync
	Sync() error
}

type syncStore struct {
	storeOpts           store.Options
	syncOpts            Options
	pendingWrites       []*deque.Deque
	pendingWriteTickers []*time.Ticker
	sync.RWMutex
}

// NewSync returns a new Sync
func NewSync(opts ...Option) Sync {
	c := &syncStore{}
	for _, o := range opts {
		o(&c.syncOpts)
	}
	if c.syncOpts.SyncInterval == 0 {
		c.syncOpts.SyncInterval = 1 * time.Minute
	}
	if c.syncOpts.SyncMultiplier == 0 {
		c.syncOpts.SyncMultiplier = 5
	}
	return c
}

func (c *syncStore) Close() error {
	return nil
}

// Init initialises the storeOptions
func (c *syncStore) Init(opts ...store.Option) error {
	for _, o := range opts {
		o(&c.storeOpts)
	}
	if len(c.syncOpts.Stores) == 0 {
		return errors.New("the sync has no stores")
	}
	if c.storeOpts.Context == nil {
		return errors.New("please provide a context to the sync. Cancelling the context signals that the sync is being disposed and syncs the sync")
	}
	for _, s := range c.syncOpts.Stores {
		if err := s.Init(); err != nil {
			return errors.Wrapf(err, "Store %s failed to Init()", s.String())
		}
	}
	c.pendingWrites = make([]*deque.Deque, len(c.syncOpts.Stores)-1)
	c.pendingWriteTickers = make([]*time.Ticker, len(c.syncOpts.Stores)-1)
	for i := 0; i < len(c.pendingWrites); i++ {
		c.pendingWrites[i] = deque.New()
		c.pendingWrites[i].Init()
		c.pendingWriteTickers[i] = time.NewTicker(c.syncOpts.SyncInterval * time.Duration(intpow(c.syncOpts.SyncMultiplier, int64(i))))
	}
	go c.syncManager()
	return nil
}

// Options returns the sync's store options
func (c *syncStore) Options() store.Options {
	return c.storeOpts
}

// String returns a printable string describing the sync
func (c *syncStore) String() string {
	backends := make([]string, len(c.syncOpts.Stores))
	for i, s := range c.syncOpts.Stores {
		backends[i] = s.String()
	}
	return fmt.Sprintf("sync %v", backends)
}

func (c *syncStore) List(opts ...store.ListOption) ([]string, error) {
	return c.syncOpts.Stores[0].List(opts...)
}

func (c *syncStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	return c.syncOpts.Stores[0].Read(key, opts...)
}

func (c *syncStore) Write(r *store.Record, opts ...store.WriteOption) error {
	return c.syncOpts.Stores[0].Write(r, opts...)
}

// Delete removes a key from the sync
func (c *syncStore) Delete(key string, opts ...store.DeleteOption) error {
	return c.syncOpts.Stores[0].Delete(key, opts...)
}

func (c *syncStore) Sync() error {
	return nil
}

type internalRecord struct {
	key       string
	value     []byte
	expiresAt time.Time
}
