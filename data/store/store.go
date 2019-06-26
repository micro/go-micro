// Package store is an interface for distribute data storage.
package store

import (
	"errors"
	"time"

	"github.com/micro/go-micro/config/options"
)

var (
	ErrNotFound = errors.New("not found")
)

// Store is a data storage interface
type Store interface {
	// embed options
	options.Options
	// Dump the known records
	Dump() ([]*Record, error)
	// Read a record with key
	Read(key string) (*Record, error)
	// Write a record
	Write(r *Record) error
	// Delete a record with key
	Delete(key string) error
}

// Record represents a data record
type Record struct {
	Key    string
	Value  []byte
	Expiry time.Duration
}
