// Package consul is a consul implementation of kv
package consul

import (
	"fmt"
	"net"

	"github.com/hashicorp/consul/api"
	"github.com/micro/go-micro/sync/data"
)

type ckv struct {
	client *api.Client
}

func (c *ckv) Read(key string) (*data.Record, error) {
	keyval, _, err := c.client.KV().Get(key, nil)
	if err != nil {
		return nil, err
	}

	if keyval == nil {
		return nil, data.ErrNotFound
	}

	return &data.Record{
		Key:   keyval.Key,
		Value: keyval.Value,
	}, nil
}

func (c *ckv) Delete(key string) error {
	_, err := c.client.KV().Delete(key, nil)
	return err
}

func (c *ckv) Write(record *data.Record) error {
	_, err := c.client.KV().Put(&api.KVPair{
		Key:   record.Key,
		Value: record.Value,
	}, nil)
	return err
}

func (c *ckv) Dump() ([]*data.Record, error) {
	keyval, _, err := c.client.KV().List("/", nil)
	if err != nil {
		return nil, err
	}
	if keyval == nil {
		return nil, data.ErrNotFound
	}
	var vals []*data.Record
	for _, keyv := range keyval {
		vals = append(vals, &data.Record{
			Key:   keyv.Key,
			Value: keyv.Value,
		})
	}
	return vals, nil
}

func (c *ckv) String() string {
	return "consul"
}

func NewData(opts ...data.Option) data.Data {
	var options data.Options
	for _, o := range opts {
		o(&options)
	}

	config := api.DefaultConfig()

	// set host
	// config.Host something
	// check if there are any addrs
	if len(options.Nodes) > 0 {
		addr, port, err := net.SplitHostPort(options.Nodes[0])
		if ae, ok := err.(*net.AddrError); ok && ae.Err == "missing port in address" {
			port = "8500"
			config.Address = fmt.Sprintf("%s:%s", options.Nodes[0], port)
		} else if err == nil {
			config.Address = fmt.Sprintf("%s:%s", addr, port)
		}
	}

	client, _ := api.NewClient(config)

	return &ckv{
		client: client,
	}
}
