// Package stats provides runtime stats
package stats

// Stats provides stats interface
type Stats interface {
	// Read a stat snapshot
	Read() (*Stat, error)
	// Write a stat snapshot
	Write(*Stat) error
}

// A runtime stat
type Stat struct {
	// Start time as unix timestamp
	Started int64
	// Uptime in nanoseconds
	Uptime int64
	// Memory usage in bytes
	Memory uint64
	// Threads aka go routines
	Threads uint64
	// Garbage collection in nanoseconds
	GC uint64
}
