// Package memory is a in-memory data store
package memory

import (
	"sync"
	"time"

	"github.com/micro/go-micro/data"
	"github.com/micro/go-micro/options"
)

type memoryData struct {
	options.Options

	sync.RWMutex
	values map[string]*memoryRecord
}

type memoryRecord struct {
	r *data.Record
	c time.Time
}

func (m *memoryData) Dump() ([]*data.Record, error) {
	m.RLock()
	defer m.RUnlock()

	var values []*data.Record

	for _, v := range m.values {
		// get expiry
		d := v.r.Expiry
		t := time.Since(v.c)

		// expired
		if d > time.Duration(0) && t > d {
			continue
		}
		values = append(values, v.r)
	}

	return values, nil
}

func (m *memoryData) Read(key string) (*data.Record, error) {
	m.RLock()
	defer m.RUnlock()

	v, ok := m.values[key]
	if !ok {
		return nil, data.ErrNotFound
	}

	// get expiry
	d := v.r.Expiry
	t := time.Since(v.c)

	// expired
	if d > time.Duration(0) && t > d {
		return nil, data.ErrNotFound
	}

	return v.r, nil
}

func (m *memoryData) Write(r *data.Record) error {
	m.Lock()
	defer m.Unlock()

	// set the record
	m.values[r.Key] = &memoryRecord{
		r: r,
		c: time.Now(),
	}

	return nil
}

func (m *memoryData) Delete(key string) error {
	m.Lock()
	defer m.Unlock()

	// delete the value
	delete(m.values, key)

	return nil
}

// NewData returns a new data.Data
func NewData(opts ...options.Option) data.Data {
	options := options.NewOptions(opts...)

	return &memoryData{
		Options: options,
		values:  make(map[string]*memoryRecord),
	}
}
