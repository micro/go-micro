// Package etcd implements a go-micro/v2/store with etcd
package etcd

import (
	"bytes"
	"context"
	"encoding/gob"
	"math"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/namespace"
	"github.com/micro/go-micro/v2/store"
	"github.com/pkg/errors"
)

type etcdStore struct {
	options store.Options

	client *clientv3.Client
	config clientv3.Config
}

// NewStore returns a new etcd store
func NewStore(opts ...store.Option) store.Store {
	e := &etcdStore{}
	for _, o := range opts {
		o(&e.options)
	}
	e.init()
	return e
}

func (e *etcdStore) Close() error {
	return e.client.Close()
}

func (e *etcdStore) Init(opts ...store.Option) error {
	for _, o := range opts {
		o(&e.options)
	}
	return e.init()
}

func (e *etcdStore) init() error {
	// ensure context is non-nil
	e.options.Context = context.Background()
	// set up config
	e.config = clientv3.Config{}
	e.applyConfig(&e.config)
	if len(e.options.Nodes) == 0 {
		e.config.Endpoints = []string{"http://127.0.0.1:2379"}
	} else {
		e.config.Endpoints = make([]string, len(e.options.Nodes))
		copy(e.config.Endpoints, e.options.Nodes)
	}
	if e.client != nil {
		e.client.Close()
	}
	client, err := clientv3.New(e.config)
	if err != nil {
		return err
	}
	e.client = client
	ns := ""
	if len(e.options.Table) > 0 {
		ns = e.options.Table
	}
	if len(e.options.Database) > 0 {
		ns = e.options.Database + "/" + ns
	}
	if len(ns) > 0 {
		e.client.KV = namespace.NewKV(e.client.KV, ns)
		e.client.Watcher = namespace.NewWatcher(e.client.Watcher, ns)
		e.client.Lease = namespace.NewLease(e.client.Lease, ns)
	}

	return nil
}

func (e *etcdStore) Options() store.Options {
	return e.options
}

func (e *etcdStore) String() string {
	return "etcd"
}

func (e *etcdStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	readOpts := store.ReadOptions{}
	for _, o := range opts {
		o(&readOpts)
	}
	if readOpts.Suffix {
		return e.readSuffix(key, readOpts)
	}

	var etcdOpts []clientv3.OpOption
	if readOpts.Prefix {
		etcdOpts = append(etcdOpts, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
	}
	resp, err := e.client.KV.Get(context.Background(), key, etcdOpts...)
	if err != nil {
		return nil, err
	}
	if resp.Count == 0 && !(readOpts.Prefix || readOpts.Suffix) {
		return nil, store.ErrNotFound
	}
	var records []*store.Record
	for _, kv := range resp.Kvs {
		ir := internalRecord{}
		if err := gob.NewDecoder(bytes.NewReader(kv.Value)).Decode(&ir); err != nil {
			return records, errors.Wrapf(err, "couldn't decode %s into internalRecord", err.Error())
		}
		r := store.Record{
			Key:   ir.Key,
			Value: ir.Value,
		}
		if !ir.ExpiresAt.IsZero() {
			r.Expiry = time.Until(ir.ExpiresAt)
		}
		records = append(records, &r)
	}
	if readOpts.Limit > 0 || readOpts.Offset > 0 {
		return records[readOpts.Offset:min(readOpts.Limit, uint(len(records)))], nil
	}
	return records, nil
}

func (e *etcdStore) readSuffix(key string, readOpts store.ReadOptions) ([]*store.Record, error) {
	opts := []store.ListOption{store.ListSuffix(key)}
	if readOpts.Prefix {
		opts = append(opts, store.ListPrefix(key))
	}
	keys, err := e.List(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "Couldn't list with suffix %s", key)
	}
	var records []*store.Record
	for _, k := range keys {
		resp, err := e.client.KV.Get(context.Background(), k)
		if err != nil {
			return nil, errors.Wrapf(err, "Couldn't get key %s", k)
		}
		ir := internalRecord{}
		if err := gob.NewDecoder(bytes.NewReader(resp.Kvs[0].Value)).Decode(&ir); err != nil {
			return records, errors.Wrapf(err, "couldn't decode %s into internalRecord", err.Error())
		}
		r := store.Record{
			Key:   ir.Key,
			Value: ir.Value,
		}
		if !ir.ExpiresAt.IsZero() {
			r.Expiry = time.Until(ir.ExpiresAt)
		}
		records = append(records, &r)

	}
	if readOpts.Limit > 0 || readOpts.Offset > 0 {
		return records[readOpts.Offset:min(readOpts.Limit, uint(len(records)))], nil
	}
	return records, nil
}

func (e *etcdStore) Write(r *store.Record, opts ...store.WriteOption) error {
	options := store.WriteOptions{}
	for _, o := range opts {
		o(&options)
	}

	if len(opts) > 0 {
		// Copy the record before applying options, or the incoming record will be mutated
		newRecord := store.Record{}
		newRecord.Key = r.Key
		newRecord.Value = make([]byte, len(r.Value))
		copy(newRecord.Value, r.Value)
		newRecord.Expiry = r.Expiry

		if !options.Expiry.IsZero() {
			newRecord.Expiry = time.Until(options.Expiry)
		}
		if options.TTL != 0 {
			newRecord.Expiry = options.TTL
		}
		return e.write(&newRecord)
	}
	return e.write(r)
}

func (e *etcdStore) write(r *store.Record) error {
	var putOpts []clientv3.OpOption
	ir := &internalRecord{}
	ir.Key = r.Key
	ir.Value = make([]byte, len(r.Value))
	copy(ir.Value, r.Value)
	if r.Expiry != 0 {
		ir.ExpiresAt = time.Now().Add(r.Expiry)
		var leasexpiry int64
		if r.Expiry.Seconds() < 5.0 {
			// minimum etcd lease is 5 seconds
			leasexpiry = 5
		} else {
			leasexpiry = int64(math.Ceil(r.Expiry.Seconds()))
		}
		lr, err := e.client.Lease.Grant(context.Background(), leasexpiry)
		if err != nil {
			return errors.Wrapf(err, "couldn't grant an etcd lease for %s", r.Key)
		}
		putOpts = append(putOpts, clientv3.WithLease(lr.ID))
	}
	b := &bytes.Buffer{}
	if err := gob.NewEncoder(b).Encode(ir); err != nil {
		return errors.Wrapf(err, "couldn't encode %s", r.Key)
	}
	_, err := e.client.KV.Put(context.Background(), ir.Key, string(b.Bytes()), putOpts...)
	return errors.Wrapf(err, "couldn't put key %s in to etcd", err)
}

func (e *etcdStore) Delete(key string, opts ...store.DeleteOption) error {
	options := store.DeleteOptions{}
	for _, o := range opts {
		o(&options)
	}
	_, err := e.client.KV.Delete(context.Background(), key)
	return errors.Wrapf(err, "couldn't delete key %s", key)
}

func (e *etcdStore) List(opts ...store.ListOption) ([]string, error) {
	options := store.ListOptions{}
	for _, o := range opts {
		o(&options)
	}
	searchPrefix := ""
	if len(options.Prefix) > 0 {
		searchPrefix = options.Prefix
	}
	resp, err := e.client.KV.Get(context.Background(), searchPrefix, clientv3.WithPrefix(), clientv3.WithKeysOnly(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't list, etcd get failed")
	}
	if len(options.Suffix) == 0 {
		keys := make([]string, resp.Count)
		for i, kv := range resp.Kvs {
			keys[i] = string(kv.Key)
		}
		return keys, nil
	}
	keys := []string{}
	for _, kv := range resp.Kvs {
		if strings.HasSuffix(string(kv.Key), options.Suffix) {
			keys = append(keys, string(kv.Key))
		}
	}
	if options.Limit > 0 || options.Offset > 0 {
		return keys[options.Offset:min(options.Limit, uint(len(keys)))], nil
	}
	return keys, nil
}

type internalRecord struct {
	Key       string
	Value     []byte
	ExpiresAt time.Time
}

func min(i, j uint) uint {
	if i < j {
		return i
	}
	return j
}
