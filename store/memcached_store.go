package store

import (
	"errors"

	mc "github.com/bradfitz/gomemcache/memcache"
)

type MemcacheStore struct {
	Client *mc.Client
}

func (m *MemcacheStore) Get(key string) (Item, error) {
	kv, err := m.Client.Get(key)
	if err != nil && err == mc.ErrCacheMiss {
		return nil, errors.New("key not found")
	} else if err != nil {
		return nil, err
	}

	if kv == nil {
		return nil, errors.New("key not found")
	}

	return &MemcacheItem{
		key:   kv.Key,
		value: kv.Value,
	}, nil
}

func (m *MemcacheStore) Del(key string) error {
	return m.Client.Delete(key)
}

func (m *MemcacheStore) Put(item Item) error {
	return m.Client.Set(&mc.Item{
		Key:   item.Key(),
		Value: item.Value(),
	})
}

func (m *MemcacheStore) NewItem(key string, value []byte) Item {
	return &MemcacheItem{
		key:   key,
		value: value,
	}
}

func NewMemcacheStore(addrs []string, opts ...Options) Store {
	if len(addrs) == 0 {
		addrs = []string{"127.0.0.1:11211"}
	}
	return &MemcacheStore{
		Client: mc.New(addrs...),
	}
}
