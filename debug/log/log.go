// Package log provides debug logging
package log

import (
	"time"
)

var (
	// DefaultLog logger
	DefaultLog = NewLog()
	// DefaultLevel is default log level
	DefaultLevel = LevelInfo
	// prefix for all messages
	prefix string
)

// Log is event log
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
	Timestamp time.Time `json:"time"`
	// Value contains log entry
	Value interface{} `json:"value"`
	// Metadata to enrich log record
	Metadata map[string]string `json:"metadata"`
}

type Stream interface {
	Chan() <-chan Record
	Stop() error
}
