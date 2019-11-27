// Package buffer provides a simple ring buffer for storing local data
package buffer

import (
	"sync"
	"time"
)

type Buffer struct {
	size int
	sync.RWMutex
	vals []*Entry
}

type Entry struct {
	Value     interface{}
	Timestamp time.Time
}

func (b *Buffer) Put(v interface{}) {
	b.Lock()
	defer b.Unlock()

	// append to values
	b.vals = append(b.vals, &Entry{
		Value:     v,
		Timestamp: time.Now(),
	})

	// trim if bigger than size required
	if len(b.vals) > b.size {
		b.vals = b.vals[1:]
	}
}

// Get returns the last n entries
func (b *Buffer) Get(n int) []*Entry {
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

// Return the entries since a specific time
func (b *Buffer) Since(t time.Time) []*Entry {
	b.RLock()
	defer b.RUnlock()

	// return all the values
	if t.IsZero() {
		return b.vals
	}

	// if its in the future return nothing
	if time.Since(t).Seconds() < 0.0 {
		return nil
	}

	for i, v := range b.vals {
		// find the starting point
		d := v.Timestamp.Sub(t)

		// return the values
		if d.Seconds() > 0.0 {
			return b.vals[i:]
		}
	}

	return nil
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
