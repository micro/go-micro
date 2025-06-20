// Package natsjskv is a go-micro store plugin for NATS JetStream Key-Value store.
package natsjskv

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/cornelk/hashmap"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"go-micro.dev/v5/store"
)

var (
	// ErrBucketNotFound is returned when the requested bucket does not exist.
	ErrBucketNotFound = errors.New("Bucket (database) not found")
)

// KeyValueEnvelope is the data structure stored in the key value store.
type KeyValueEnvelope struct {
	Key      string                 `json:"key"`
	Data     []byte                 `json:"data"`
	Metadata map[string]interface{} `json:"metadata"`
}

type natsStore struct {
	sync.Once
	sync.RWMutex

	encoding    string
	ttl         time.Duration
	storageType nats.StorageType
	description string

	opts      store.Options
	nopts     nats.Options
	jsopts    []nats.JSOpt
	kvConfigs []*nats.KeyValueConfig

	conn    *nats.Conn
	js      nats.JetStreamContext
	buckets *hashmap.Map[string, nats.KeyValue]
}

// NewStore will create a new NATS JetStream Object Store.
func NewStore(opts ...store.Option) store.Store {
	options := store.Options{
		Nodes:    []string{},
		Database: "default",
		Table:    "",
		Context:  context.Background(),
	}

	n := &natsStore{
		description: "KeyValue storage administered by go-micro store plugin",
		opts:        options,
		jsopts:      []nats.JSOpt{},
		kvConfigs:   []*nats.KeyValueConfig{},
		buckets:     hashmap.New[string, nats.KeyValue](),
		storageType: nats.FileStorage,
	}

	n.setOption(opts...)

	return n
}

// Init initializes the store. It must perform any required setup on the
// backing storage implementation and check that it is ready for use,
// returning any errors.
func (n *natsStore) Init(opts ...store.Option) error {
	n.setOption(opts...)

	// Connect to NATS servers
	conn, err := n.nopts.Connect()
	if err != nil {
		return errors.Wrap(err, "Failed to connect to NATS Server")
	}

	// Create JetStream context
	js, err := conn.JetStream(n.jsopts...)
	if err != nil {
		return errors.Wrap(err, "Failed to create JetStream context")
	}

	n.conn = conn
	n.js = js

	// Create default config if no configs present
	if len(n.kvConfigs) == 0 {
		if _, err := n.mustGetBucketByName(n.opts.Database); err != nil {
			return err
		}
	}

	// Create kv store buckets
	for _, cfg := range n.kvConfigs {
		if _, err := n.mustGetBucket(cfg); err != nil {
			return err
		}
	}

	return nil
}

func (n *natsStore) setOption(opts ...store.Option) {
	for _, o := range opts {
		o(&n.opts)
	}

	n.Once.Do(func() {
		n.nopts = nats.GetDefaultOptions()
	})

	// Extract options from context
	if nopts, ok := n.opts.Context.Value(natsOptionsKey{}).(nats.Options); ok {
		n.nopts = nopts
	}

	if jsopts, ok := n.opts.Context.Value(jsOptionsKey{}).([]nats.JSOpt); ok {
		n.jsopts = append(n.jsopts, jsopts...)
	}

	if cfg, ok := n.opts.Context.Value(kvOptionsKey{}).([]*nats.KeyValueConfig); ok {
		n.kvConfigs = append(n.kvConfigs, cfg...)
	}

	if ttl, ok := n.opts.Context.Value(ttlOptionsKey{}).(time.Duration); ok {
		n.ttl = ttl
	}

	if sType, ok := n.opts.Context.Value(memoryOptionsKey{}).(nats.StorageType); ok {
		n.storageType = sType
	}

	if text, ok := n.opts.Context.Value(descriptionOptionsKey{}).(string); ok {
		n.description = text
	}

	if encoding, ok := n.opts.Context.Value(keyEncodeOptionsKey{}).(string); ok {
		n.encoding = encoding
	}

	// Assign store option server addresses to nats options
	if len(n.opts.Nodes) > 0 {
		n.nopts.Url = ""
		n.nopts.Servers = n.opts.Nodes
	}

	if len(n.nopts.Servers) == 0 && n.nopts.Url == "" {
		n.nopts.Url = nats.DefaultURL
	}
}

// Options allows you to view the current options.
func (n *natsStore) Options() store.Options {
	return n.opts
}

// Read takes a single key name and optional ReadOptions. It returns matching []*Record or an error.
func (n *natsStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	if err := n.initConn(); err != nil {
		return nil, err
	}

	opt := store.ReadOptions{}

	for _, o := range opts {
		o(&opt)
	}

	if opt.Database == "" {
		opt.Database = n.opts.Database
	}

	if opt.Table == "" {
		opt.Table = n.opts.Table
	}

	bucket, ok := n.buckets.Get(opt.Database)
	if !ok {
		return nil, ErrBucketNotFound
	}

	keys, err := n.natsKeys(bucket, opt.Table, key, opt.Prefix, opt.Suffix)
	if err != nil {
		return nil, err
	}

	records := make([]*store.Record, 0, len(keys))

	for _, key := range keys {
		rec, ok, err := n.getRecord(bucket, key)
		if err != nil {
			return nil, err
		}

		if ok {
			records = append(records, rec)
		}
	}

	return enforceLimits(records, opt.Limit, opt.Offset), nil
}

// Write writes a record to the store, and returns an error if the record was not written.
func (n *natsStore) Write(rec *store.Record, opts ...store.WriteOption) error {
	if err := n.initConn(); err != nil {
		return err
	}

	opt := store.WriteOptions{}
	for _, o := range opts {
		o(&opt)
	}

	if opt.Database == "" {
		opt.Database = n.opts.Database
	}

	if opt.Table == "" {
		opt.Table = n.opts.Table
	}

	store, err := n.mustGetBucketByName(opt.Database)
	if err != nil {
		return err
	}

	b, err := json.Marshal(KeyValueEnvelope{
		Key:      rec.Key,
		Data:     rec.Value,
		Metadata: rec.Metadata,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to marshal object")
	}

	if _, err := store.Put(n.NatsKey(opt.Table, rec.Key), b); err != nil {
		return errors.Wrapf(err, "Failed to store data in bucket '%s'", n.NatsKey(opt.Table, rec.Key))
	}

	return nil
}

// Delete removes the record with the corresponding key from the store.
func (n *natsStore) Delete(key string, opts ...store.DeleteOption) error {
	if err := n.initConn(); err != nil {
		return err
	}

	opt := store.DeleteOptions{}

	for _, o := range opts {
		o(&opt)
	}

	if opt.Database == "" {
		opt.Database = n.opts.Database
	}

	if opt.Table == "" {
		opt.Table = n.opts.Table
	}

	if opt.Table == "DELETE_BUCKET" {
		n.buckets.Del(key)

		if err := n.js.DeleteKeyValue(key); err != nil {
			return errors.Wrap(err, "Failed to delete bucket")
		}

		return nil
	}

	store, ok := n.buckets.Get(opt.Database)
	if !ok {
		return ErrBucketNotFound
	}

	if err := store.Delete(n.NatsKey(opt.Table, key)); err != nil {
		return errors.Wrap(err, "Failed to delete data")
	}

	return nil
}

// List returns any keys that match, or an empty list with no error if none matched.
func (n *natsStore) List(opts ...store.ListOption) ([]string, error) {
	if err := n.initConn(); err != nil {
		return nil, err
	}

	opt := store.ListOptions{}
	for _, o := range opts {
		o(&opt)
	}

	if opt.Database == "" {
		opt.Database = n.opts.Database
	}

	if opt.Table == "" {
		opt.Table = n.opts.Table
	}

	store, ok := n.buckets.Get(opt.Database)
	if !ok {
		return nil, ErrBucketNotFound
	}

	keys, err := n.microKeys(store, opt.Table, opt.Prefix, opt.Suffix)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list keys in bucket")
	}

	return enforceLimits(keys, opt.Limit, opt.Offset), nil
}

// Close the store.
func (n *natsStore) Close() error {
	n.conn.Close()
	return nil
}

// String returns the name of the implementation.
func (n *natsStore) String() string {
	return "NATS JetStream KeyValueStore"
}

// thread safe way to initialize the connection.
func (n *natsStore) initConn() error {
	if n.hasConn() {
		return nil
	}

	n.Lock()
	defer n.Unlock()

	// check if conn was initialized meanwhile
	if n.conn != nil {
		return nil
	}

	return n.Init()
}

// thread safe way to check if n is initialized.
func (n *natsStore) hasConn() bool {
	n.RLock()
	defer n.RUnlock()

	return n.conn != nil
}

// mustGetDefaultBucket returns the bucket with the given name creating it with default configuration if needed.
func (n *natsStore) mustGetBucketByName(name string) (nats.KeyValue, error) {
	return n.mustGetBucket(&nats.KeyValueConfig{
		Bucket:      name,
		Description: n.description,
		TTL:         n.ttl,
		Storage:     n.storageType,
	})
}

// mustGetBucket creates a new bucket if it does not exist yet.
func (n *natsStore) mustGetBucket(kv *nats.KeyValueConfig) (nats.KeyValue, error) {
	if store, ok := n.buckets.Get(kv.Bucket); ok {
		return store, nil
	}

	store, err := n.js.KeyValue(kv.Bucket)
	if err != nil {
		if !errors.Is(err, nats.ErrBucketNotFound) {
			return nil, errors.Wrapf(err, "Failed to get bucket (%s)", kv.Bucket)
		}

		store, err = n.js.CreateKeyValue(kv)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create bucket (%s)", kv.Bucket)
		}
	}

	n.buckets.Set(kv.Bucket, store)

	return store, nil
}

// getRecord returns the record with the given key from the nats kv store.
func (n *natsStore) getRecord(bucket nats.KeyValue, key string) (*store.Record, bool, error) {
	obj, err := bucket.Get(key)
	if errors.Is(err, nats.ErrKeyNotFound) {
		return nil, false, store.ErrNotFound
	} else if err != nil {
		return nil, false, errors.Wrap(err, "Failed to get object from bucket")
	}

	var kv KeyValueEnvelope
	if err := json.Unmarshal(obj.Value(), &kv); err != nil {
		return nil, false, errors.Wrap(err, "Failed to unmarshal object")
	}

	if obj.Operation() != nats.KeyValuePut {
		return nil, false, nil
	}

	return &store.Record{
		Key:      kv.Key,
		Value:    kv.Data,
		Metadata: kv.Metadata,
	}, true, nil
}

func (n *natsStore) natsKeys(bucket nats.KeyValue, table, key string, prefix, suffix bool) ([]string, error) {
	if !suffix && !prefix {
		return []string{n.NatsKey(table, key)}, nil
	}

	toS := func(s string, b bool) string {
		if b {
			return s
		}

		return ""
	}

	keys, _, err := n.getKeys(bucket, table, toS(key, prefix), toS(key, suffix))

	return keys, err
}

func (n *natsStore) microKeys(bucket nats.KeyValue, table, prefix, suffix string) ([]string, error) {
	_, keys, err := n.getKeys(bucket, table, prefix, suffix)

	return keys, err
}

func (n *natsStore) getKeys(bucket nats.KeyValue, table string, prefix, suffix string) ([]string, []string, error) {
	names, err := bucket.Keys(nats.IgnoreDeletes())
	if errors.Is(err, nats.ErrKeyNotFound) {
		return []string{}, []string{}, nil
	} else if err != nil {
		return []string{}, []string{}, errors.Wrap(err, "Failed to list objects")
	}

	natsKeys := make([]string, 0, len(names))
	microKeys := make([]string, 0, len(names))

	for _, k := range names {
		mkey, ok := n.MicroKeyFilter(table, k, prefix, suffix)
		if !ok {
			continue
		}

		natsKeys = append(natsKeys, k)
		microKeys = append(microKeys, mkey)
	}

	return natsKeys, microKeys, nil
}

// enforces offset and limit without causing a panic.
func enforceLimits[V any](recs []V, limit, offset uint) []V {
	l := uint(len(recs))

	from := offset
	if from > l {
		from = l
	}

	to := l
	if limit > 0 && offset+limit < l {
		to = offset + limit
	}

	return recs[from:to]
}
