// Package log provides a logging interface
package log

import (
	"time"
)

// Log provides access to logs
type Log interface {
	Read(...ReadOption) ([]*Entry, error)
	Write([]*Entry, ...WriteOption) error
	String() string
}

// A single log entry
type Entry struct {
	Id       string
	Time     time.Time
	Message  []byte
	Metadata map[string]string
}

type ReadOption func(o *ReadOptions)

type WriteOption func(o *WriteOptions)

type ReadOptions struct {
	// read the given id
	Id string
	// Number of entries to read
	Entries int
	// Filter function
	Filter func(*Entry) bool
}

type WriteOptions struct{}
