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
	// Sync all the known records
	Sync() ([]*Record, error)
	// Read a record with key
	Read(keys ...string) ([]*Record, error)
	// Write a record
	Write(recs ...*Record) error
	// Delete a record with key
	Delete(keys ...string) error
}

// Record represents a data record
type Record struct {
	Key    string
	Value  []byte
	Expiry time.Duration
}
