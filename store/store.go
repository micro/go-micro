// Package store is an interface for distribute data storage.
package store

import (
	"errors"
	"time"
)

var (
	// ErrNotFound is returned when a Read key doesn't exist
	ErrNotFound = errors.New("not found")
	// Default store
	DefaultStore Store = new(noop)
)

// Store is a data storage interface
type Store interface {
	// Initialise store options
	Init(...Option) error
	// List all the known records
	List() ([]*Record, error)
	// Read records with keys
	Read(key string, opts ...ReadOption) ([]*Record, error)
	// Write records
	Write(*Record) error
	// Delete records with keys
	Delete(key string) error
	// Name of the store
	String() string
}

// Record represents a data record
type Record struct {
	Key    string
	Value  []byte
	Expiry time.Duration
}

type ReadOptions struct {
	// Read key as a prefix
	Prefix bool
	// Read key as a suffix
	Suffix bool
}

type ReadOption func(o *ReadOptions)

type noop struct{}

func (n *noop) Init(...Option) error {
	return nil
}

func (n *noop) List() ([]*Record, error) {
	return nil, nil
}

func (n *noop) Read(key string, opts ...ReadOption) ([]*Record, error) {
	return nil, nil
}

func (n *noop) Write(rec *Record) error {
	return nil
}

func (n *noop) Delete(key string) error {
	return nil
}

func (n *noop) String() string {
	return "noop"
}
