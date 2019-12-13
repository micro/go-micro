// Package etcd is an etcd v3 implementation of kv
package etcd

import (
	"context"
	"log"

	client "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/store"
)

type ekv struct {
	options.Options
	kv client.KV
}

func (e *ekv) Read(keys ...string) ([]*store.Record, error) {
	//nolint:prealloc
	var values []*mvccpb.KeyValue

	for _, key := range keys {
		keyval, err := e.kv.Get(context.Background(), key)
		if err != nil {
			return nil, err
		}

		if keyval == nil || len(keyval.Kvs) == 0 {
			return nil, store.ErrNotFound
		}

		values = append(values, keyval.Kvs...)
	}

	records := make([]*store.Record, 0, len(values))

	for _, kv := range values {
		records = append(records, &store.Record{
			Key:   string(kv.Key),
			Value: kv.Value,
			// TODO: implement expiry
		})
	}

	return records, nil
}

func (e *ekv) Delete(keys ...string) error {
	var gerr error
	for _, key := range keys {
		_, err := e.kv.Delete(context.Background(), key)
		if err != nil {
			gerr = err
		}
	}
	return gerr
}

func (e *ekv) Write(records ...*store.Record) error {
	var gerr error
	for _, record := range records {
		// TODO create lease to expire keys
		_, err := e.kv.Put(context.Background(), record.Key, string(record.Value))
		if err != nil {
			gerr = err
		}
	}
	return gerr
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

func NewStore(opts ...options.Option) store.Store {
	options := options.NewOptions(opts...)

	var endpoints []string

	if e, ok := options.Values().Get("store.nodes"); ok {
		endpoints = e.([]string)
	}

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
		Options: options,
		kv:      client.NewKV(c),
	}
}
