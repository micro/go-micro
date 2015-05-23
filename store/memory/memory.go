package memory

import (
	"errors"
	"sync"

	"github.com/myodc/go-micro/store"
)

type mstore struct {
	sync.RWMutex
	store map[string]store.Item
}

type item struct {
	key   string
	value []byte
}

func (i *item) Key() string {
	return i.key
}

func (i *item) Value() []byte {
	return i.value
}

func (m *mstore) Get(key string) (store.Item, error) {
	m.RLock()
	v, ok := m.store[key]
	m.RUnlock()
	if !ok {
		return nil, errors.New("key not found")
	}
	return v, nil
}

func (m *mstore) Del(key string) error {
	m.Lock()
	delete(m.store, key)
	m.Unlock()
	return nil
}

func (m *mstore) Put(item store.Item) error {
	m.Lock()
	m.store[item.Key()] = item
	m.Unlock()
	return nil
}

func (m *mstore) NewItem(key string, value []byte) store.Item {
	return &item{
		key:   key,
		value: value,
	}
}

func NewStore(addrs []string, opt ...store.Option) store.Store {
	return &mstore{
		store: make(map[string]store.Item),
	}
}
