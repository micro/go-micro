// Package memory is a in-memory store store
package memory

import (
	"strings"
	"time"

	"github.com/micro/go-micro/v2/store"

	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
)

// NewStore returns a memory store
func NewStore(opts ...store.Option) store.Store {
	s := &memoryStore{
		options: store.Options{},
		store:   cache.New(cache.NoExpiration, 5*time.Minute),
	}
	for _, o := range opts {
		o(&s.options)
	}
	return s
}

type memoryStore struct {
	options store.Options

	store *cache.Cache
}

func (m *memoryStore) Init(opts ...store.Option) error {
	m.store.Flush()
	for _, o := range opts {
		o(&m.options)
	}
	return nil
}

func (m *memoryStore) Options() store.Options {
	return m.options
}

func (m *memoryStore) String() string {
	return "memory"
}

func (m *memoryStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	readOpts := store.ReadOptions{}
	for _, o := range opts {
		o(&readOpts)
	}
	originalKey := key
	if len(m.options.Prefix) > 0 {
		key = m.options.Prefix + "/" + key
	}
	if len(m.options.Namespace) > 0 {
		key = m.options.Namespace + "/" + key
	}

	var keys []string

	if readOpts.Prefix || readOpts.Suffix {
		var opts []store.ListOption
		if readOpts.Prefix {
			opts = append(opts, store.ListPrefix(originalKey))
		}
		if readOpts.Suffix {
			opts = append(opts, store.ListSuffix(originalKey))
		}
		k, err := m.List(opts...)
		if err != nil {
			return nil, errors.Wrap(err, "Memory: Read couldn't List()")
		}
		keys = k
	} else {
		keys = []string{key}
	}
	var results []*store.Record
	for _, k := range keys {
		r, err := m.get(k)
		if err != nil {
			return results, err
		}
		results = append(results, r)
	}

	return results, nil
}

func (m *memoryStore) get(k string) (*store.Record, error) {
	if len(m.options.Prefix) > 0 {
		k = m.options.Prefix + "/" + k
	}
	if len(m.options.Namespace) > 0 {
		k = m.options.Namespace + "/" + k
	}
	var storedRecord *store.Record
	r, found := m.store.Get(k)
	if !found {
		return nil, store.ErrNotFound
	}
	storedRecord, ok := r.(*store.Record)
	if !ok {
		return nil, errors.New("Retrieved a non *store.Record from the cache")
	}
	// Copy the record on the way out
	newRecord := &store.Record{}
	newRecord.Key = storedRecord.Key
	newRecord.Value = make([]byte, len(storedRecord.Value))
	copy(newRecord.Value, storedRecord.Value)
	newRecord.Expiry = storedRecord.Expiry

	return newRecord, nil
}

func (m *memoryStore) Write(r *store.Record, opts ...store.WriteOption) error {
	writeOpts := store.WriteOptions{}
	for _, o := range opts {
		o(&writeOpts)
	}
	key := r.Key
	if len(m.options.Prefix) > 0 {
		key = m.options.Prefix + "/" + key
	}
	if len(m.options.Namespace) > 0 {
		key = m.options.Namespace + "/" + key
	}

	// Copy the incoming record and store it
	newRecord := store.Record{}
	newRecord.Key = r.Key
	newRecord.Value = make([]byte, len(r.Value))
	copy(newRecord.Value, r.Value)
	newRecord.Expiry = r.Expiry

	if !writeOpts.Expiry.IsZero() {
		newRecord.Expiry = time.Until(writeOpts.Expiry)
	}
	if writeOpts.TTL != 0 {
		newRecord.Expiry = writeOpts.TTL
	}

	m.store.Set(key, &newRecord, newRecord.Expiry)
	return nil
}

func (m *memoryStore) Delete(key string, opts ...store.DeleteOption) error {
	deleteOptions := store.DeleteOptions{}
	for _, o := range opts {
		o(&deleteOptions)
	}
	if len(m.options.Prefix) > 0 {
		key = m.options.Prefix + "/" + key
	}
	if len(m.options.Namespace) > 0 {
		key = m.options.Namespace + "/" + key
	}

	m.store.Delete(key)
	return nil
}

func (m *memoryStore) List(opts ...store.ListOption) ([]string, error) {
	listOptions := store.ListOptions{}
	for _, o := range opts {
		o(&listOptions)
	}
	allKeys := m.list()
	for i := range allKeys {
		if len(m.options.Namespace) > 0 {
			allKeys[i] = strings.TrimPrefix(allKeys[i], m.options.Namespace+"/")
		}
		if len(m.options.Prefix) > 0 {
			allKeys[i] = strings.TrimPrefix(allKeys[i], m.options.Prefix+"/")
		}
	}
	if len(listOptions.Prefix) > 0 {
		var prefixKeys []string
		for _, k := range allKeys {
			if strings.HasPrefix(k, listOptions.Prefix) {
				prefixKeys = append(prefixKeys, k)
			}
		}
		allKeys = prefixKeys
	}
	if len(listOptions.Suffix) > 0 {
		var suffixKeys []string
		for _, k := range allKeys {
			if strings.HasPrefix(k, listOptions.Prefix) {
				suffixKeys = append(suffixKeys, k)
			}
		}
		allKeys = suffixKeys
	}

	return allKeys, nil
}

func (m *memoryStore) list() []string {
	allItems := m.store.Items()
	allKeys := make([]string, len(allItems))
	i := 0
	for k := range allItems {
		allKeys[i] = k
		i++
	}
	return allKeys
}
