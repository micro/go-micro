// Package consul is a consul implementation of kv
package consul

import (
	"fmt"
	"log"
	"net"
	"path/filepath"

	"github.com/hashicorp/consul/api"
	"github.com/asim/go-micro/v3/store"
)

type ckv struct {
	options store.Options
	client  *api.Client
}

func (c *ckv) Init(opts ...store.Option) error {
	for _, o := range opts {
		o(&c.options)
	}
	return c.configure()
}

func (c *ckv) Options() store.Options {
	return c.options
}

func (c *ckv) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	// TODO: implement read options
	records := make([]*store.Record, 0, 1)

	keyval, _, err := c.client.KV().Get(key, nil)
	if err != nil {
		return nil, err
	}

	if keyval == nil {
		return nil, store.ErrNotFound
	}

	records = append(records, &store.Record{
		Key:   keyval.Key,
		Value: keyval.Value,
	})

	return records, nil
}

func (c *ckv) Close() error {
	// consul does not support close?
	return nil
}

func (c *ckv) Delete(key string, opts ...store.DeleteOption) error {
	options := store.DeleteOptions{}
	options.Table = c.options.Table

	for _, o := range opts {
		o(&options)
	}

	_, err := c.client.KV().Delete(filepath.Join(options.Table, key), nil)
	return err
}

func (c *ckv) Write(record *store.Record, opts ...store.WriteOption) error {
	options := store.WriteOptions{}
	options.Table = c.options.Table

	for _, o := range opts {
		o(&options)
	}

	_, err := c.client.KV().Put(&api.KVPair{
		Key:   filepath.Join(options.Table, record.Key),
		Value: record.Value,
	}, nil)
	return err
}

func (c *ckv) List(opts ...store.ListOption) ([]string, error) {
	options := store.ListOptions{}
	options.Table = c.options.Table

	for _, o := range opts {
		o(&options)
	}
	if options.Table == "" {
		options.Table = "/"
	} else if options.Table[len(options.Table)-1] != '/' {
		options.Table += "/"
	}

	keyval, _, err := c.client.KV().List(options.Table, nil)
	if err != nil {
		return nil, err
	}
	if keyval == nil {
		return nil, store.ErrNotFound
	}
	var keys []string
	for _, keyv := range keyval {
		keys = append(keys, keyv.Key)
	}
	return keys, nil
}

func (c *ckv) String() string {
	return "consul"
}

func NewStore(opts ...store.Option) store.Store {
	var options store.Options
	for _, o := range opts {
		o(&options)
	}

	// new store
	s := new(ckv)
	// set the options
	s.options = options

	// configure the store
	if err := s.configure(); err != nil {
		log.Fatal(err)
	}

	// return store
	return s
}

func (ckv *ckv) configure() error {
	config := api.DefaultConfig()
	nodes := ckv.options.Nodes

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

	client, err := api.NewClient(config)
	if err != nil {
		return err
	}

	ckv.client = client

	return nil
}
