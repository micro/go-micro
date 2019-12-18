// Package stats provides runtime stats
package stats

// Stats provides stats interface
type Stats interface {
	// Read stat snapshot
	Read() ([]*Stat, error)
	// Write a stat snapshot
	Write(*Stat) error
	// Record a request
	Record(error) error
}

// A runtime stat
type Stat struct {
	// Timestamp of recording
	Timestamp int64
	// Start time as unix timestamp
	Started int64
	// Uptime in seconds
	Uptime int64
	// Memory usage in bytes
	Memory uint64
	// Threads aka go routines
	Threads uint64
	// Garbage collection in nanoseconds
	GC uint64
	// Total requests
	Requests uint64
	// Total errors
	Errors uint64
}

var (
	DefaultStats = NewStats()
)
