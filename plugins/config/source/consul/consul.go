package consul

import (
	"fmt"
	"net"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/asim/go-micro/v3/config/source"
)

// Currently a single consul reader
type consul struct {
	prefix      string
	stripPrefix string
	addr        string
	opts        source.Options
	client      *api.Client
}

var (
	// DefaultPrefix is the prefix that consul keys will be assumed to have if you
	// haven't specified one
	DefaultPrefix = "/micro/config/"
)

func (c *consul) Read() (*source.ChangeSet, error) {
	kv, _, err := c.client.KV().List(c.prefix, nil)
	if err != nil {
		return nil, err
	}

	if kv == nil || len(kv) == 0 {
		return nil, fmt.Errorf("source not found: %s", c.prefix)
	}

	data, err := makeMap(c.opts.Encoder, kv, c.stripPrefix)
	if err != nil {
		return nil, fmt.Errorf("error reading data: %v", err)
	}

	b, err := c.opts.Encoder.Encode(data)
	if err != nil {
		return nil, fmt.Errorf("error reading source: %v", err)
	}

	cs := &source.ChangeSet{
		Timestamp: time.Now(),
		Format:    c.opts.Encoder.String(),
		Source:    c.String(),
		Data:      b,
	}
	cs.Checksum = cs.Sum()

	return cs, nil
}

// Write is unsupported
func (c *consul) Write(cs *source.ChangeSet) error {
	return nil
}

func (c *consul) String() string {
	return "consul"
}

func (c *consul) Watch() (source.Watcher, error) {
	w, err := newWatcher(c.prefix, c.addr, c.String(), c.stripPrefix, c.opts.Encoder)
	if err != nil {
		return nil, err
	}
	return w, nil
}

// NewSource creates a new consul source
func NewSource(opts ...source.Option) source.Source {
	options := source.NewOptions(opts...)

	// use default config
	config := api.DefaultConfig()

	// use the consul config passed in the options if any
	if co, ok := options.Context.Value(configKey{}).(*api.Config); ok {
		config = co
	}

	// check if there are any addrs
	a, ok := options.Context.Value(addressKey{}).(string)
	if ok {
		addr, port, err := net.SplitHostPort(a)
		if ae, ok := err.(*net.AddrError); ok && ae.Err == "missing port in address" {
			port = "8500"
			addr = a
			config.Address = fmt.Sprintf("%s:%s", addr, port)
		} else if err == nil {
			config.Address = fmt.Sprintf("%s:%s", addr, port)
		}
	}

	dc, ok := options.Context.Value(dcKey{}).(string)
	if ok {
		config.Datacenter = dc
	}

	token, ok := options.Context.Value(tokenKey{}).(string)
	if ok {
		config.Token = token
	}

	// create the client
	client, _ := api.NewClient(config)

	prefix := DefaultPrefix
	sp := ""
	f, ok := options.Context.Value(prefixKey{}).(string)
	if ok {
		prefix = f
	}

	if b, ok := options.Context.Value(stripPrefixKey{}).(bool); ok && b {
		sp = prefix
	}

	return &consul{
		prefix:      prefix,
		stripPrefix: sp,
		addr:        config.Address,
		opts:        options,
		client:      client,
	}
}
