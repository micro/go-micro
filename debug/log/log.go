// Package log provides debug logging
package log

import (
	"time"
)

var (
	// Default buffer size if any
	DefaultSize = 1024
	// DefaultLog logger
	DefaultLog = NewLog()
)

// Log is debug log interface for reading and writing logs
type Log interface {
	// Read reads log entries from the logger
	Read(...ReadOption) ([]Record, error)
	// Write writes records to log
	Write(Record) error
	// Stream log records
	Stream() (Stream, error)
}

// Record is log record entry
type Record struct {
	// Timestamp of logged event
	Timestamp time.Time `json:"timestamp"`
	// Metadata to enrich log record
	Metadata map[string]string `json:"metadata"`
	// Value contains log entry
	Message interface{} `json:"message"`
}

// Stream returns a log stream
type Stream interface {
	Chan() <-chan Record
	Stop() error
}
