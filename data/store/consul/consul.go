// Package consul is a consul implementation of kv
package consul

import (
	"fmt"
	"net"

	"github.com/hashicorp/consul/api"
	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/data/store"
)

type ckv struct {
	options.Options
	client *api.Client
}

func (c *ckv) Read(key string) (*store.Record, error) {
	keyval, _, err := c.client.KV().Get(key, nil)
	if err != nil {
		return nil, err
	}

	if keyval == nil {
		return nil, store.ErrNotFound
	}

	return &store.Record{
		Key:   keyval.Key,
		Value: keyval.Value,
	}, nil
}

func (c *ckv) Delete(key string) error {
	_, err := c.client.KV().Delete(key, nil)
	return err
}

func (c *ckv) Write(record *store.Record) error {
	_, err := c.client.KV().Put(&api.KVPair{
		Key:   record.Key,
		Value: record.Value,
	}, nil)
	return err
}

func (c *ckv) Dump() ([]*store.Record, error) {
	keyval, _, err := c.client.KV().List("/", nil)
	if err != nil {
		return nil, err
	}
	if keyval == nil {
		return nil, store.ErrNotFound
	}
	var vals []*store.Record
	for _, keyv := range keyval {
		vals = append(vals, &store.Record{
			Key:   keyv.Key,
			Value: keyv.Value,
		})
	}
	return vals, nil
}

func (c *ckv) String() string {
	return "consul"
}

func NewStore(opts ...options.Option) store.Store {
	options := options.NewOptions(opts...)
	config := api.DefaultConfig()

	var nodes []string

	if n, ok := options.Values().Get("store.nodes"); ok {
		nodes = n.([]string)
	}

	// set host
	if len(nodes) > 0 {
		addr, port, err := net.SplitHostPort(nodes[0])
		if ae, ok := err.(*net.AddrError); ok && ae.Err == "missing port in address" {
			port = "8500"
			config.Address = fmt.Sprintf("%s:%s", nodes[0], port)
		} else if err == nil {
			config.Address = fmt.Sprintf("%s:%s", addr, port)
		}
	}

	client, _ := api.NewClient(config)

	return &ckv{
		Options: options,
		client:  client,
	}
}
