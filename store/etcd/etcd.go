// Package etcd is an etcd v3 implementation of kv
package etcd

import (
	"context"
	"log"

	client "github.com/coreos/etcd/clientv3"
	"github.com/micro/go-micro/v2/store"
)

type ekv struct {
	options store.Options
	kv      client.KV
}

func (e *ekv) Init(opts ...store.Option) error {
	for _, o := range opts {
		o(&e.options)
	}
	return nil
}

func (e *ekv) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	var options store.ReadOptions
	for _, o := range opts {
		o(&options)
	}

	var etcdOpts []client.OpOption

	// set options prefix
	if options.Prefix {
		etcdOpts = append(etcdOpts, client.WithPrefix())
	}

	keyval, err := e.kv.Get(context.Background(), key, etcdOpts...)
	if err != nil {
		return nil, err
	}

	if keyval == nil || len(keyval.Kvs) == 0 {
		return nil, store.ErrNotFound
	}

	records := make([]*store.Record, 0, len(keyval.Kvs))

	for _, kv := range keyval.Kvs {
		records = append(records, &store.Record{
			Key:   string(kv.Key),
			Value: kv.Value,
			// TODO: implement expiry
		})
	}

	return records, nil
}

func (e *ekv) Delete(key string) error {
	_, err := e.kv.Delete(context.Background(), key)
	return err
}

func (e *ekv) Write(record *store.Record) error {
	// TODO create lease to expire keys
	_, err := e.kv.Put(context.Background(), record.Key, string(record.Value))
	return err
}

func (e *ekv) List() ([]*store.Record, error) {
	keyval, err := e.kv.Get(context.Background(), "/", client.WithPrefix())
	if err != nil {
		return nil, err
	}
	if keyval == nil || len(keyval.Kvs) == 0 {
		return nil, nil
	}
	vals := make([]*store.Record, 0, len(keyval.Kvs))
	for _, keyv := range keyval.Kvs {
		vals = append(vals, &store.Record{
			Key:   string(keyv.Key),
			Value: keyv.Value,
		})
	}
	return vals, nil
}

func (e *ekv) String() string {
	return "etcd"
}

func NewStore(opts ...store.Option) store.Store {
	var options store.Options
	for _, o := range opts {
		o(&options)
	}

	// get the endpoints
	endpoints := options.Nodes

	if len(endpoints) == 0 {
		endpoints = []string{"http://127.0.0.1:2379"}
	}

	// TODO: parse addresses
	c, err := client.New(client.Config{
		Endpoints: endpoints,
	})
	if err != nil {
		log.Fatal(err)
	}

	return &ekv{
		options: options,
		kv:      client.NewKV(c),
	}
}
