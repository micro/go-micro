// Package etcd is an etcd v3 implementation of kv
package etcd

import (
	"context"
	"log"

	client "github.com/coreos/etcd/clientv3"
	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/store"
)

type ekv struct {
	options.Options
	kv client.KV
}

func (e *ekv) Read(key string) (*store.Record, error) {
	keyval, err := e.kv.Get(context.Background(), key)
	if err != nil {
		return nil, err
	}

	if keyval == nil || len(keyval.Kvs) == 0 {
		return nil, store.ErrNotFound
	}

	return &store.Record{
		Key:   string(keyval.Kvs[0].Key),
		Value: keyval.Kvs[0].Value,
	}, nil
}

func (e *ekv) Delete(key string) error {
	_, err := e.kv.Delete(context.Background(), key)
	return err
}

func (e *ekv) Write(record *store.Record) error {
	_, err := e.kv.Put(context.Background(), record.Key, string(record.Value))
	return err
}

func (e *ekv) Dump() ([]*store.Record, error) {
	keyval, err := e.kv.Get(context.Background(), "/", client.WithPrefix())
	if err != nil {
		return nil, err
	}
	var vals []*store.Record
	if keyval == nil || len(keyval.Kvs) == 0 {
		return vals, nil
	}
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
