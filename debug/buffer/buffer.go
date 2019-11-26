// Package buffer provides a simple ring buffer for storing local data
package buffer

import (
	"sync"
)

type Buffer struct {
	size int
	sync.RWMutex
	vals []interface{}
}

func (b *Buffer) Put(v interface{}) {
	b.Lock()
	defer b.Unlock()

	// append to values
	b.vals = append(b.vals, v)

	// trim if bigger than size required
	if len(b.vals) > b.size {
		b.vals = b.vals[1:]
	}
}

// Get returns the last n entries
func (b *Buffer) Get(n int) []interface{} {
	// reset any invalid values
	if n > b.size || n < 0 {
		n = b.size
	}

	b.RLock()
	defer b.RUnlock()

	// create a delta
	delta := b.size - n

	// if all the values are less than delta
	if len(b.vals) < delta {
		return b.vals
	}

	// return the delta set
	return b.vals[delta:]
}

func (b *Buffer) Size() int {
	return b.size
}

// New returns a new buffer of the given size
func New(i int) *Buffer {
	return &Buffer{
		size: i,
	}
}
