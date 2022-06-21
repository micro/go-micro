package cache

import (
	"context"
	"sync"
	"time"
)

type memCache struct {
	opts Options
	sync.RWMutex

	items map[string]Item
}

func (c *memCache) Get(ctx context.Context, key string) (interface{}, time.Time, error) {
	c.RWMutex.RLock()
	defer c.RWMutex.RUnlock()

	item, found := c.items[key]
	if !found {
		return nil, time.Time{}, ErrKeyNotFound
	}
	if item.Expired() {
		return nil, time.Time{}, ErrItemExpired
	}

	return item.Value, time.Unix(0, item.Expiration), nil
}

func (c *memCache) Put(ctx context.Context, key string, val interface{}, d time.Duration) error {
	var e int64
	if d == DefaultExpiration {
		d = c.opts.Expiration
	}
	if d > 0 {
		e = time.Now().Add(d).UnixNano()
	}

	c.RWMutex.Lock()
	defer c.RWMutex.Unlock()

	c.items[key] = Item{
		Value:      val,
		Expiration: e,
	}

	return nil
}

func (c *memCache) Delete(ctx context.Context, key string) error {
	c.RWMutex.Lock()
	defer c.RWMutex.Unlock()

	_, found := c.items[key]
	if !found {
		return ErrKeyNotFound
	}

	delete(c.items, key)
	return nil
}

func (m *memCache) String() string {
	return "memory"
}
