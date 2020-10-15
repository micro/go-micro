// Package memory is an in memory cache
package memory

import (
	"sync"

	"github.com/micro/go-micro/v3/cache"
	"github.com/micro/go-micro/v3/errors"
)

type memoryCache struct {
	// TODO: use a decent caching library
	sync.RWMutex
	values map[string]interface{}
}

func (m *memoryCache) Init(opts ...cache.Option) error {
	// TODO: implement
	return nil
}

func (m *memoryCache) Get(key string) (interface{}, error) {
	m.RLock()
	defer m.RUnlock()

	v, ok := m.values[key]
	if !ok {
		return nil, errors.NotFound("go.micro.cache", key+" not found")
	}

	return v, nil
}

func (m *memoryCache) Set(key string, val interface{}) error {
	m.Lock()
	m.values[key] = val
	m.Unlock()
	return nil
}

func (m *memoryCache) Delete(key string) error {
	m.Lock()
	delete(m.values, key)
	m.Unlock()
	return nil
}

func (m *memoryCache) String() string {
	return "memory"
}

func NewCache(opts ...cache.Option) cache.Cache {
	return &memoryCache{
		values: make(map[string]interface{}),
	}
}
