// Package data is an interface for distribute data storage.
package data

import (
	"errors"
	"time"

	"github.com/micro/go-micro/options"
)

var (
	ErrNotFound = errors.New("not found")
)

// Data is a data storage interface
type Data interface {
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
