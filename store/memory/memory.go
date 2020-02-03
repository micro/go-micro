// Package memory is a in-memory store store
package memory

import (
	"strings"
	"sync"
	"time"

	"github.com/micro/go-micro/v2/store"
)

type memoryStore struct {
	options store.Options

	sync.RWMutex
	values map[string]*memoryRecord
}

type memoryRecord struct {
	r *store.Record
	c time.Time
}

func (m *memoryStore) Init(opts ...store.Option) error {
	for _, o := range opts {
		o(&m.options)
	}
	return nil
}

func (m *memoryStore) List() ([]*store.Record, error) {
	m.RLock()
	defer m.RUnlock()

	//nolint:prealloc
	var values []*store.Record

	for _, v := range m.values {
		// get expiry
		d := v.r.Expiry
		t := time.Since(v.c)

		if d > time.Duration(0) {
			// expired
			if t > d {
				continue
			}
			// update expiry
			v.r.Expiry -= t
			v.c = time.Now()
		}

		values = append(values, v.r)
	}

	return values, nil
}

func (m *memoryStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	m.RLock()
	defer m.RUnlock()

	var options store.ReadOptions

	for _, o := range opts {
		o(&options)
	}

	var vals []*memoryRecord

	if options.Prefix {
		for _, v := range m.values {
			if !strings.HasPrefix(v.r.Key, key) {
				continue
			}
			vals = append(vals, v)
		}
	} else if options.Suffix {
		for _, v := range m.values {
			if !strings.HasSuffix(v.r.Key, key) {
				continue
			}
			vals = append(vals, v)
		}
	} else {
		v, ok := m.values[key]
		if !ok {
			return nil, store.ErrNotFound
		}
		vals = []*memoryRecord{v}
	}

	//nolint:prealloc
	var records []*store.Record

	for _, v := range vals {
		// get expiry
		d := v.r.Expiry
		t := time.Since(v.c)

		// expired
		if d > time.Duration(0) {
			if t > d {
				return nil, store.ErrNotFound
			}
			// update expiry
			v.r.Expiry -= t
			v.c = time.Now()
		}

		records = append(records, v.r)
	}

	return records, nil
}

func (m *memoryStore) Write(r *store.Record) error {
	m.Lock()
	defer m.Unlock()

	// set the record
	m.values[r.Key] = &memoryRecord{
		r: r,
		c: time.Now(),
	}

	return nil
}

func (m *memoryStore) Delete(key string) error {
	m.Lock()
	defer m.Unlock()

	// delete the value
	delete(m.values, key)

	return nil
}

func (m *memoryStore) String() string {
	return "memory"
}

// NewStore returns a new store.Store
func NewStore(opts ...store.Option) store.Store {
	var options store.Options
	for _, o := range opts {
		o(&options)
	}

	return &memoryStore{
		options: options,
		values:  make(map[string]*memoryRecord),
	}
}
