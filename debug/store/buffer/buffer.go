// Package buffer provides a simple ring buffer
package buffer

import (
	"sync"

	"github.com/micro/go-micro/debug/store"
)

type Buffer struct {
	sync.RWMutex
	index   map[string]int
	records []*store.Record
}

func (b *Buffer) Read(opts ...store.ReadOption) ([]*store.Record, error) {
	var options store.ReadOptions
	for _, o := range opts {
		o(&options)
	}

	b.RLock()
	defer b.RUnlock()

	if len(options.Id) > 0 {
		idx, ok := b.index[options.Id]
		if !ok || len(b.records) < idx {
			return nil, store.ErrNotFound
		}
		return []*store.Record{b.records[idx]}, nil
	}

	return b.records, nil
}

func (b *Buffer) Write(records []*store.Record) error {
	b.Lock()
	defer b.Unlock()

	if b.index == nil {
		b.index = make(map[string]int)
	}

	i := len(b.records)

	for _, r := range records {
		b.index[r.Id] = i
		i++
		b.records = append(b.records, r)
	}

	return nil
}

func (b *Buffer) String() string {
	return "buffer"
}

type Record struct {
	Id    string
	Value interface{}
}
