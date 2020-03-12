// Package memory is a in-memory store store
package memory

import (
	"sort"
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

func (m *memoryStore) String() string {
	return "memory"
}

func (m *memoryStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	readOpts := store.ReadOptions{}
	for _, o := range opts {
		o(&readOpts)
	}

	var keys []string

	// Handle Prefix / suffix
	if readOpts.Prefix || readOpts.Suffix {
		var opts []store.ListOption
		if readOpts.Prefix {
			opts = append(opts, store.ListPrefix(key))
		}
		if readOpts.Suffix {
			opts = append(opts, store.ListSuffix(key))
		}
		opts = append(opts, store.ListLimit(readOpts.Limit))
		opts = append(opts, store.ListOffset(readOpts.Offset))
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
	if len(m.options.Suffix) > 0 {
		k = k + m.options.Suffix
	}
	if len(m.options.Prefix) > 0 {
		k = m.options.Prefix + "/" + k
	}
	if len(m.options.Namespace) > 0 {
		k = m.options.Namespace + "/" + k
	}
	var storedRecord *internalRecord
	r, found := m.store.Get(k)
	if !found {
		return nil, store.ErrNotFound
	}
	storedRecord, ok := r.(*internalRecord)
	if !ok {
		return nil, errors.New("Retrieved a non *internalRecord from the cache")
	}
	// Copy the record on the way out
	newRecord := &store.Record{}
	newRecord.Key = storedRecord.key
	newRecord.Value = make([]byte, len(storedRecord.value))
	copy(newRecord.Value, storedRecord.value)
	newRecord.Expiry = time.Until(storedRecord.expiresAt)

	return newRecord, nil
}

func (m *memoryStore) Write(r *store.Record, opts ...store.WriteOption) error {
	writeOpts := store.WriteOptions{}
	for _, o := range opts {
		o(&writeOpts)
	}

	if len(opts) > 0 {
		// Copy the record before applying options, or the incoming record will be mutated
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
		m.set(&newRecord)
	} else {
		m.set(r)
	}
	return nil
}

func (m *memoryStore) set(r *store.Record) {
	key := r.Key
	if len(m.options.Suffix) > 0 {
		key = key + m.options.Suffix
	}
	if len(m.options.Prefix) > 0 {
		key = m.options.Prefix + "/" + key
	}
	if len(m.options.Namespace) > 0 {
		key = m.options.Namespace + "/" + key
	}

	// copy the incoming record and then
	// convert the expiry in to a hard timestamp
	i := &internalRecord{}
	i.key = r.Key
	i.value = make([]byte, len(r.Value))
	copy(i.value, r.Value)
	if r.Expiry != 0 {
		i.expiresAt = time.Now().Add(r.Expiry)
	}

	m.store.Set(key, i, r.Expiry)
}

func (m *memoryStore) Delete(key string, opts ...store.DeleteOption) error {
	deleteOptions := store.DeleteOptions{}
	for _, o := range opts {
		o(&deleteOptions)
	}
	m.delete(key)
	return nil
}

func (m *memoryStore) delete(key string) {
	if len(m.options.Suffix) > 0 {
		key = key + m.options.Suffix
	}
	if len(m.options.Prefix) > 0 {
		key = m.options.Prefix + "/" + key
	}
	if len(m.options.Namespace) > 0 {
		key = m.options.Namespace + "/" + key
	}
	m.store.Delete(key)
}

func (m *memoryStore) Options() store.Options {
	return m.options
}

func (m *memoryStore) List(opts ...store.ListOption) ([]string, error) {
	listOptions := store.ListOptions{}

	for _, o := range opts {
		o(&listOptions)
	}
	allKeys := m.list(listOptions.Limit, listOptions.Offset)

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
			if strings.HasSuffix(k, listOptions.Suffix) {
				suffixKeys = append(suffixKeys, k)
			}
		}
		allKeys = suffixKeys
	}

	return allKeys, nil
}

func (m *memoryStore) list(limit, offset uint) []string {
	allItems := m.store.Items()
	allKeys := make([]string, len(allItems))
	i := 0
	for k := range allItems {
		if len(m.options.Suffix) > 0 {
			k = strings.TrimSuffix(k, m.options.Suffix)
		}
		if len(m.options.Namespace) > 0 {
			k = strings.TrimPrefix(k, m.options.Namespace+"/")
		}
		if len(m.options.Prefix) > 0 {
			k = strings.TrimPrefix(k, m.options.Prefix+"/")
		}
		allKeys[i] = k
		i++
	}
	if limit != 0 || offset != 0 {
		sort.Slice(allKeys, func(i, j int) bool { return allKeys[i] < allKeys[j] })
		min := func(i, j uint) uint {
			if i < j {
				return i
			}
			return j
		}
		return allKeys[offset:min(limit, uint(len(allKeys)))]
	}
	return allKeys
}

type internalRecord struct {
	key       string
	value     []byte
	expiresAt time.Time
}
