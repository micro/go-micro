// Package memory is a in-memory store store
package memory

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/asim/nitro/v3/store"
	"github.com/patrickmn/go-cache"
)

// NewStore returns a memory store
func NewStore(opts ...store.Option) store.Store {
	s := &memoryStore{
		options: store.Options{
			Database: "micro",
			Table:    "micro",
		},
		stores: map[string]*cache.Cache{}, // cache.New(cache.NoExpiration, 5*time.Minute),
	}
	for _, o := range opts {
		o(&s.options)
	}
	return s
}

type memoryStore struct {
	sync.RWMutex
	options store.Options

	stores map[string]*cache.Cache
}

type storeRecord struct {
	key       string
	value     []byte
	metadata  map[string]interface{}
	expiresAt time.Time
}

func (m *memoryStore) prefix(database, table string) string {
	if len(database) == 0 {
		database = m.options.Database
	}
	if len(table) == 0 {
		table = m.options.Table
	}
	return filepath.Join(database, table)
}

func (m *memoryStore) getStore(prefix string) *cache.Cache {
	m.RLock()
	store := m.stores[prefix]
	m.RUnlock()
	if store == nil {
		m.Lock()
		if m.stores[prefix] == nil {
			m.stores[prefix] = cache.New(cache.NoExpiration, 5*time.Minute)
		}
		store = m.stores[prefix]
		m.Unlock()
	}
	return store
}

func (m *memoryStore) get(prefix, key string) (*store.Record, error) {
	var storedRecord *storeRecord
	r, found := m.getStore(prefix).Get(key)
	if !found {
		return nil, store.ErrNotFound
	}

	storedRecord, ok := r.(*storeRecord)
	if !ok {
		return nil, errors.New("Retrieved a non *storeRecord from the cache")
	}

	// Copy the record on the way out
	newRecord := &store.Record{}
	newRecord.Key = strings.TrimPrefix(storedRecord.key, prefix+"/")
	newRecord.Value = make([]byte, len(storedRecord.value))
	newRecord.Metadata = make(map[string]interface{})

	// copy the value into the new record
	copy(newRecord.Value, storedRecord.value)

	// check if we need to set the expiry
	if !storedRecord.expiresAt.IsZero() {
		newRecord.Expiry = time.Until(storedRecord.expiresAt)
	}

	// copy in the metadata
	for k, v := range storedRecord.metadata {
		newRecord.Metadata[k] = v
	}

	return newRecord, nil
}

func (m *memoryStore) set(prefix string, r *store.Record) {
	// copy the incoming record and then
	// convert the expiry in to a hard timestamp
	i := &storeRecord{}
	i.key = r.Key
	i.value = make([]byte, len(r.Value))
	i.metadata = make(map[string]interface{})

	// copy the the value
	copy(i.value, r.Value)

	// set the expiry
	if r.Expiry != 0 {
		i.expiresAt = time.Now().Add(r.Expiry)
	}

	// set the metadata
	for k, v := range r.Metadata {
		i.metadata[k] = v
	}

	m.getStore(prefix).Set(r.Key, i, r.Expiry)
}

func (m *memoryStore) delete(prefix, key string) {
	m.getStore(prefix).Delete(key)
}

func (m *memoryStore) list(prefix string, limit, offset uint, prefixFilter, suffixFilter string) []string {

	allItems := m.getStore(prefix).Items()

	allKeys := make([]string, len(allItems))

	// construct list of keys for this prefix
	i := 0
	for k := range allItems {
		allKeys[i] = k
		i++
	}
	keys := make([]string, 0, len(allKeys))
	sort.Slice(allKeys, func(i, j int) bool { return allKeys[i] < allKeys[j] })
	for _, k := range allKeys {
		if prefixFilter != "" && !strings.HasPrefix(k, prefixFilter) {
			continue
		}
		if suffixFilter != "" && !strings.HasSuffix(k, suffixFilter) {
			continue
		}
		if offset > 0 {
			offset--
			continue
		}
		keys = append(keys, k)
		// this check still works if no limit was passed to begin with, you'll just end up with large -ve value
		if limit == 1 {
			break
		}
		limit--
	}
	return keys
}

func (m *memoryStore) Close() error {
	m.Lock()
	defer m.Unlock()
	for _, s := range m.stores {
		s.Flush()
	}
	return nil
}

func (m *memoryStore) Init(opts ...store.Option) error {
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

	prefix := m.prefix(readOpts.Database, readOpts.Table)

	var keys []string
	// Handle Prefix / suffix
	if readOpts.Prefix || readOpts.Suffix {
		prefixFilter := ""
		if readOpts.Prefix {
			prefixFilter = key
		}
		suffixFilter := ""
		if readOpts.Suffix {
			suffixFilter = key
		}
		keys = m.list(prefix, readOpts.Limit, readOpts.Offset, prefixFilter, suffixFilter)
	} else {
		keys = []string{key}
	}

	var results []*store.Record

	for _, k := range keys {
		r, err := m.get(prefix, k)
		if err != nil {
			return results, err
		}
		results = append(results, r)
	}

	return results, nil
}

func (m *memoryStore) Write(r *store.Record, opts ...store.WriteOption) error {
	writeOpts := store.WriteOptions{}
	for _, o := range opts {
		o(&writeOpts)
	}

	prefix := m.prefix(writeOpts.Database, writeOpts.Table)

	if len(opts) > 0 {
		// Copy the record before applying options, or the incoming record will be mutated
		newRecord := store.Record{}
		newRecord.Key = r.Key
		newRecord.Value = make([]byte, len(r.Value))
		newRecord.Metadata = make(map[string]interface{})
		copy(newRecord.Value, r.Value)
		newRecord.Expiry = r.Expiry

		for k, v := range r.Metadata {
			newRecord.Metadata[k] = v
		}

		m.set(prefix, &newRecord)
		return nil
	}

	// set
	m.set(prefix, r)

	return nil
}

func (m *memoryStore) Delete(key string, opts ...store.DeleteOption) error {
	deleteOptions := store.DeleteOptions{}
	for _, o := range opts {
		o(&deleteOptions)
	}

	prefix := m.prefix(deleteOptions.Database, deleteOptions.Table)
	m.delete(prefix, key)
	return nil
}

func (m *memoryStore) Options() store.Options {
	return m.options
}

func (m *memoryStore) List(opts ...store.ListOption) ([]string, error) {
	listOptions := store.ListOptions{}

	for _, o := range opts {
		o(&listOptions)
	}

	prefix := m.prefix(listOptions.Database, listOptions.Table)
	keys := m.list(prefix, listOptions.Limit, listOptions.Offset, listOptions.Prefix, listOptions.Suffix)
	return keys, nil
}
