package memcached

import (
	"errors"

	mc "github.com/bradfitz/gomemcache/memcache"
	"github.com/myodc/go-micro/store"
)

type mstore struct {
	Client *mc.Client
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
	kv, err := m.Client.Get(key)
	if err != nil && err == mc.ErrCacheMiss {
		return nil, errors.New("key not found")
	} else if err != nil {
		return nil, err
	}

	if kv == nil {
		return nil, errors.New("key not found")
	}

	return &item{
		key:   kv.Key,
		value: kv.Value,
	}, nil
}

func (m *mstore) Del(key string) error {
	return m.Client.Delete(key)
}

func (m *mstore) Put(item store.Item) error {
	return m.Client.Set(&mc.Item{
		Key:   item.Key(),
		Value: item.Value(),
	})
}

func (m *mstore) NewItem(key string, value []byte) store.Item {
	return &item{
		key:   key,
		value: value,
	}
}

func NewStore(addrs []string, opts ...store.Option) store.Store {
	if len(addrs) == 0 {
		addrs = []string{"127.0.0.1:11211"}
	}
	return &mstore{
		Client: mc.New(addrs...),
	}
}
