// Package memcache is a memcache implementation of the Cache
package memcache

import (
	"encoding/json"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/micro/go-micro/v3/cache"
)

type memcacheCache struct {
	options cache.Options
	client  *memcache.Client
}

type memcacheItem struct {
	Key   string
	Value interface{}
}

func (m *memcacheCache) Init(opts ...cache.Option) error {
	for _, o := range opts {
		o(&m.options)
	}
	return nil
}

func (m *memcacheCache) Get(key string) (interface{}, error) {
	item, err := m.client.Get(key)
	if err != nil {
		return nil, err
	}

	var mc *memcacheItem

	if err := json.Unmarshal(item.Value, &mc); err != nil {
		return nil, err
	}

	return mc.Value, nil
}

func (m *memcacheCache) Set(key string, val interface{}) error {
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}

	return m.client.Set(&memcache.Item{
		Key:   key,
		Value: b,
	})
}

func (m *memcacheCache) Delete(key string) error {
	return m.client.Delete(key)
}

func (m *memcacheCache) String() string {
	return "memcache"
}

// NewCache returns a new memcache Cache
func NewCache(opts ...cache.Option) cache.Cache {
	var options cache.Options
	for _, o := range opts {
		o(&options)
	}

	// get and set the nodes
	nodes := options.Nodes
	if len(nodes) == 0 {
		nodes = []string{"localhost:11211"}
	}

	return &memcacheCache{
		options: options,
		client:  memcache.New(nodes...),
	}
}
