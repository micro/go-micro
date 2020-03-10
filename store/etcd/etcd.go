// Package etcd implements a go-micro/v2/store with etcd
package etcd

import (
	"context"

	"github.com/micro/go-micro/v2/store"
	"go.etcd.io/etcd/clientv3"
)

type etcdStore struct {
	options store.Options

	kv clientv3.KV
}

// NewStore returns a new etcd store
func NewStore(opts ...store.Option) store.Store {
	e := &etcdStore{}
	for _, o := range opts {
		o(&e.options)
	}
	e.init()
	return e
}

func (e *etcdStore) Init(opts ...store.Option) error {
	for _, o := range opts {
		o(&e.options)
	}
	return e.init()
}

func (e *etcdStore) init() error {
	// ensure context is non-nil
	e.options.Context = context.Background()
	// set up config
	conf := clientv3.Config{}
	e.applyConfig(&conf)
	if len(e.options.Nodes) == 0 {
		conf.Endpoints = []string{"http://127.0.0.1:2379"}
	} else {
		conf.Endpoints = make([]string, len(e.options.Nodes))
		copy(conf.Endpoints, e.options.Nodes)
	}
	client, err := clientv3.New(conf)
	if err != nil {
		return err
	}
	e.kv = clientv3.NewKV(client)

	return nil
}

func (e *etcdStore) Options() store.Options {
	return e.options
}

func (e *etcdStore) String() string {
	return "etcd"
}

func (e *etcdStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	return nil, nil
}

func (e *etcdStore) Write(r *store.Record, opts ...store.WriteOption) error {
	return nil
}

func (e *etcdStore) Delete(key string, opts ...store.DeleteOption) error {
	return nil
}

func (e *etcdStore) List(opts ...store.ListOption) ([]string, error) {
	return nil, nil
}
