package store

import (
	"errors"

	consul "github.com/hashicorp/consul/api"
)

type consulStore struct {
	Client *consul.Client
}

func (c *consulStore) Get(key string) (Item, error) {
	kv, _, err := c.Client.KV().Get(key, nil)
	if err != nil {
		return nil, err
	}
	if kv == nil {
		return nil, errors.New("key not found")
	}

	return &consulItem{
		key:   kv.Key,
		value: kv.Value,
	}, nil
}

func (c *consulStore) Del(key string) error {
	_, err := c.Client.KV().Delete(key, nil)
	return err
}

func (c *consulStore) Put(item Item) error {
	_, err := c.Client.KV().Put(&consul.KVPair{
		Key:   item.Key(),
		Value: item.Value(),
	}, nil)

	return err
}

func (c *consulStore) NewItem(key string, value []byte) Item {
	return &consulItem{
		key:   key,
		value: value,
	}
}

func newConsulStore(addrs []string, opt ...Option) Store {
	config := consul.DefaultConfig()
	if len(addrs) > 0 {
		config.Address = addrs[0]
	}

	client, _ := consul.NewClient(config)

	return &consulStore{
		Client: client,
	}
}
