// Package data is an interface for key-value storage.
package data

import (
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("not found")
)

// Data is a data storage interface
type Data interface {
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
	Key        string
	Value      []byte
	Expiration time.Duration
}

type Option func(o *Options)
