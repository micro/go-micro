// Package etcd is an etcd v3 implementation of kv
package etcd

import (
	"context"
	"log"

	"github.com/micro/go-micro/sync/data"
	client "go.etcd.io/etcd/clientv3"
)

type ekv struct {
	kv client.KV
}

func (e *ekv) Read(key string) (*data.Record, error) {
	keyval, err := e.kv.Get(context.Background(), key)
	if err != nil {
		return nil, err
	}

	if keyval == nil || len(keyval.Kvs) == 0 {
		return nil, data.ErrNotFound
	}

	return &data.Record{
		Key:   string(keyval.Kvs[0].Key),
		Value: keyval.Kvs[0].Value,
	}, nil
}

func (e *ekv) Delete(key string) error {
	_, err := e.kv.Delete(context.Background(), key)
	return err
}

func (e *ekv) Write(record *data.Record) error {
	_, err := e.kv.Put(context.Background(), record.Key, string(record.Value))
	return err
}

func (e *ekv) Dump() ([]*data.Record, error) {
	keyval, err := e.kv.Get(context.Background(), "/", client.WithPrefix())
	if err != nil {
		return nil, err
	}
	var vals []*data.Record
	if keyval == nil || len(keyval.Kvs) == 0 {
		return vals, nil
	}
	for _, keyv := range keyval.Kvs {
		vals = append(vals, &data.Record{
			Key:   string(keyv.Key),
			Value: keyv.Value,
		})
	}
	return vals, nil
}

func (e *ekv) String() string {
	return "etcd"
}

func NewData(opts ...data.Option) data.Data {
	var options data.Options
	for _, o := range opts {
		o(&options)
	}

	var endpoints []string

	for _, addr := range options.Nodes {
		if len(addr) > 0 {
			endpoints = append(endpoints, addr)
		}
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
		kv: client.NewKV(c),
	}
}
