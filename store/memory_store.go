package store

import (
	"errors"
	"sync"
)

type MemoryStore struct {
	sync.RWMutex
	store map[string]Item
}

func (m *MemoryStore) Get(key string) (Item, error) {
	m.RLock()
	v, ok := m.store[key]
	m.RUnlock()
	if !ok {
		return nil, errors.New("key not found")
	}
	return v, nil
}

func (m *MemoryStore) Del(key string) error {
	m.Lock()
	delete(m.store, key)
	m.Unlock()
	return nil
}

func (m *MemoryStore) Put(item Item) error {
	m.Lock()
	m.store[item.Key()] = item
	m.Unlock()
	return nil
}

func (m *MemoryStore) NewItem(key string, value []byte) Item {
	return &MemoryItem{
		key:   key,
		value: value,
	}
}

func NewMemoryStore(addrs []string, opts ...Options) Store {
	return &MemoryStore{
		store: make(map[string]Item),
	}
}
