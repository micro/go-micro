package nats

import (
	"fmt"
	"net"
	"strings"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"go-micro.dev/v4/config/source"
	log "go-micro.dev/v4/logger"
)

type nats struct {
	url    string
	bucket string
	key    string
	kv     natsgo.KeyValue
	opts   source.Options
}

// DefaultBucket is the bucket that nats keys will be assumed to have if you
// haven't specified one
var (
	DefaultBucket = "default"
	DefaultKey    = "micro_config"
)

func (n *nats) Read() (*source.ChangeSet, error) {
	e, err := n.kv.Get(n.key)
	if err != nil {
		if err == natsgo.ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}

	if e.Value() == nil || len(e.Value()) == 0 {
		return nil, fmt.Errorf("source not found: %s", n.key)
	}

	cs := &source.ChangeSet{
		Data:      e.Value(),
		Format:    n.opts.Encoder.String(),
		Source:    n.String(),
		Timestamp: time.Now(),
	}
	cs.Checksum = cs.Sum()

	return cs, nil
}

func (n *nats) Write(cs *source.ChangeSet) error {
	_, err := n.kv.Put(n.key, cs.Data)
	if err != nil {
		return err
	}

	return nil
}

func (n *nats) String() string {
	return "nats"
}

func (n *nats) Watch() (source.Watcher, error) {
	return newWatcher(n.kv, n.bucket, n.key, n.String(), n.opts.Encoder)
}

func NewSource(opts ...source.Option) source.Source {
	options := source.NewOptions(opts...)

	config := natsgo.DefaultOptions

	urls, ok := options.Context.Value(urlKey{}).([]string)
	endpoints := []string{}
	if ok {
		for _, u := range urls {
			addr, port, err := net.SplitHostPort(u)
			if ae, ok := err.(*net.AddrError); ok && ae.Err == "missing port in address" {
				port = "4222"
				addr = u
				endpoints = append(endpoints, fmt.Sprintf("%s:%s", addr, port))
			} else if err == nil {
				endpoints = append(endpoints, fmt.Sprintf("%s:%s", addr, port))
			}
		}
	}
	if len(endpoints) == 0 {
		endpoints = append(endpoints, "127.0.0.1:4222")
	}

	bucket, ok := options.Context.Value(bucketKey{}).(string)
	if !ok {
		bucket = DefaultBucket
	}

	key, ok := options.Context.Value(keyKey{}).(string)
	if !ok {
		key = DefaultKey
	}

	config.Url = strings.Join(endpoints, ",")

	nc, err := natsgo.Connect(config.Url)
	if err != nil {
		log.Error(err)
	}

	js, err := nc.JetStream(natsgo.MaxWait(10 * time.Second))
	if err != nil {
		log.Error(err)
	}

	kv, err := js.KeyValue(bucket)
	if err == natsgo.ErrBucketNotFound || err == natsgo.ErrKeyNotFound {
		kv, err = js.CreateKeyValue(&natsgo.KeyValueConfig{Bucket: bucket})
		if err != nil {
			log.Error(err)
		}
	}

	if err != nil {
		log.Error(err)
	}

	return &nats{
		url:    config.Url,
		bucket: bucket,
		key:    key,
		kv:     kv,
		opts:   options,
	}
}
