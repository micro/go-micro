// Package buffer provides a simple ring buffer for storing local data
package buffer

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type stream struct {
	id      string
	entries chan *Entry
	stop    chan bool
}

// Buffer is ring buffer
type Buffer struct {
	size int
	sync.RWMutex
	vals    []*Entry
	streams map[string]stream
}

// Entry is ring buffer data entry
type Entry struct {
	Value     interface{}
	Timestamp time.Time
}

// New returns a new buffer of the given size
func New(i int) *Buffer {
	return &Buffer{
		size:    i,
		streams: make(map[string]stream),
	}
}

// Put adds a new value to ring buffer
func (b *Buffer) Put(v interface{}) {
	b.Lock()
	defer b.Unlock()

	// append to values
	entry := &Entry{
		Value:     v,
		Timestamp: time.Now(),
	}
	b.vals = append(b.vals, entry)

	// trim if bigger than size required
	if len(b.vals) > b.size {
		b.vals = b.vals[1:]
	}

	// TODO: this is fucking ugly
	for _, stream := range b.streams {
		select {
		case <-stream.stop:
			delete(b.streams, stream.id)
			close(stream.entries)
		case stream.entries <- entry:
		}
	}
}

// Get returns the last n entries
func (b *Buffer) Get(n int) []*Entry {
	b.RLock()
	defer b.RUnlock()

	// reset any invalid values
	if n > b.size || n < 0 {
		n = b.size
	}

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

// Stream logs from the buffer
func (b *Buffer) Stream(stop chan bool) <-chan *Entry {
	b.Lock()
	defer b.Unlock()

	entries := make(chan *Entry, 128)
	id := uuid.New().String()
	b.streams[id] = stream{
		id:      id,
		entries: entries,
		stop:    stop,
	}

	return entries
}

// Size returns the size of the ring buffer
func (b *Buffer) Size() int {
	return b.size
}
