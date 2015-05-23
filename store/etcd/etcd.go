package etcd

import (
	"errors"

	"github.com/coreos/go-etcd/etcd"
	"github.com/myodc/go-micro/store"
)

type estore struct {
	Client *etcd.Client
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

func (e *estore) Get(key string) (store.Item, error) {
	kv, err := e.Client.Get(key, false, false)
	if err != nil {
		return nil, err
	}
	if kv == nil {
		return nil, errors.New("key not found")
	}

	return &item{
		key:   kv.Node.Key,
		value: []byte(kv.Node.Value),
	}, nil
}

func (e *estore) Del(key string) error {
	_, err := e.Client.Delete(key, false)
	return err
}

func (e *estore) Put(item store.Item) error {
	_, err := e.Client.Set(item.Key(), string(item.Value()), 0)

	return err
}

func (e *estore) NewItem(key string, value []byte) store.Item {
	return &item{
		key:   key,
		value: value,
	}
}

func NewStore(addrs []string, opts ...store.Option) store.Store {
	if len(addrs) == 0 {
		addrs = []string{"127.0.0.1:2379"}
	}

	client := etcd.NewClient(addrs)

	return &estore{
		Client: client,
	}
}
