package store

import (
	"errors"

	"github.com/coreos/go-etcd/etcd"
)

type EtcdStore struct {
	Client *etcd.Client
}

func (e *EtcdStore) Get(key string) (Item, error) {
	kv, err := e.Client.Get(key, false, false)
	if err != nil {
		return nil, err
	}
	if kv == nil {
		return nil, errors.New("key not found")
	}

	return &EtcdItem{
		key:   kv.Node.Key,
		value: []byte(kv.Node.Value),
	}, nil
}

func (e *EtcdStore) Del(key string) error {
	_, err := e.Client.Delete(key, false)
	return err
}

func (e *EtcdStore) Put(item Item) error {
	_, err := e.Client.Set(item.Key(), string(item.Value()), 0)

	return err
}

func (e *EtcdStore) NewItem(key string, value []byte) Item {
	return &EtcdItem{
		key:   key,
		value: value,
	}
}

func NewEtcdStore(addrs []string, opts ...Options) Store {
	if len(addrs) == 0 {
		addrs = []string{"127.0.0.1:2379"}
	}

	client := etcd.NewClient(addrs)

	return &EtcdStore{
		Client: client,
	}
}
