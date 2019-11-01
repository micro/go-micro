// Package store is an interface for distribute data storage.
package store

import (
	"errors"
	"time"
)

var (
	// ErrNotFound is returned when a Read key doesn't exist
	ErrNotFound = errors.New("not found")
)

// Store is a data storage interface
type Store interface {
	// List all the known records
	List() ([]*Record, error)
	// Read records with keys
	Read(key ...string) ([]*Record, error)
	// Write records
	Write(rec ...*Record) error
	// Delete records with keys
	Delete(key ...string) error
}

// Record represents a data record
type Record struct {
	Key    string
	Value  []byte
	Expiry time.Duration
}
