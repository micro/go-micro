// Package consul is a consul implementation of kv
package consul

import (
	"fmt"
	"net"

	"github.com/hashicorp/consul/api"
	"github.com/micro/go-micro/data"
	"github.com/micro/go-micro/options"
)

type ckv struct {
	options.Options
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

func NewData(opts ...options.Option) data.Data {
	options := options.NewOptions(opts...)
	config := api.DefaultConfig()

	var nodes []string

	if n, ok := options.Values().Get("data.nodes"); ok {
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
