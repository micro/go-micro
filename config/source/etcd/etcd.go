package etcd

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/micro/go-micro/config/source"
	cetcd "go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
)

// Currently a single etcd reader
type etcd struct {
	prefix      string
	stripPrefix string
	opts        source.Options
	client      *cetcd.Client
	cerr        error
}

var (
	DefaultPrefix = "/micro/config/"
)

func (c *etcd) Read() (*source.ChangeSet, error) {
	if c.cerr != nil {
		return nil, c.cerr
	}

	rsp, err := c.client.Get(context.Background(), c.prefix, cetcd.WithPrefix())
	if err != nil {
		return nil, err
	}

	if rsp == nil || len(rsp.Kvs) == 0 {
		return nil, fmt.Errorf("source not found: %s", c.prefix)
	}

	var kvs []*mvccpb.KeyValue
	for _, v := range rsp.Kvs {
		kvs = append(kvs, (*mvccpb.KeyValue)(v))
	}

	data := makeMap(c.opts.Encoder, kvs, c.stripPrefix)

	b, err := c.opts.Encoder.Encode(data)
	if err != nil {
		return nil, fmt.Errorf("error reading source: %v", err)
	}

	cs := &source.ChangeSet{
		Timestamp: time.Now(),
		Source:    c.String(),
		Data:      b,
		Format:    c.opts.Encoder.String(),
	}
	cs.Checksum = cs.Sum()

	return cs, nil
}

func (c *etcd) String() string {
	return "etcd"
}

func (c *etcd) Watch() (source.Watcher, error) {
	if c.cerr != nil {
		return nil, c.cerr
	}
	cs, err := c.Read()
	if err != nil {
		return nil, err
	}
	return newWatcher(c.prefix, c.stripPrefix, c.client.Watcher, cs, c.opts)
}

func NewSource(opts ...source.Option) source.Source {
	options := source.NewOptions(opts...)

	var endpoints []string

	// check if there are any addrs
	addrs, ok := options.Context.Value(addressKey{}).([]string)
	if ok {
		for _, a := range addrs {
			addr, port, err := net.SplitHostPort(a)
			if ae, ok := err.(*net.AddrError); ok && ae.Err == "missing port in address" {
				port = "2379"
				addr = a
				endpoints = append(endpoints, fmt.Sprintf("%s:%s", addr, port))
			} else if err == nil {
				endpoints = append(endpoints, fmt.Sprintf("%s:%s", addr, port))
			}
		}
	}

	if len(endpoints) == 0 {
		endpoints = []string{"localhost:2379"}
	}

	config := cetcd.Config{
		Endpoints: endpoints,
	}

	u, ok := options.Context.Value(authKey{}).(*authCreds)
	if ok {
		config.Username = u.Username
		config.Password = u.Password
	}

	// use default config
	client, err := cetcd.New(config)

	prefix := DefaultPrefix
	sp := ""
	f, ok := options.Context.Value(prefixKey{}).(string)
	if ok {
		prefix = f
	}

	if b, ok := options.Context.Value(stripPrefixKey{}).(bool); ok && b {
		sp = prefix
	}

	return &etcd{
		prefix:      prefix,
		stripPrefix: sp,
		opts:        options,
		client:      client,
		cerr:        err,
	}
}
