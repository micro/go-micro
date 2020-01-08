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

type noop struct{}

func (n *noop) Init(...Option) error {
	return nil
}

func (n *noop) List() ([]*Record, error) {
	return nil, nil
}

func (n *noop) Read(key ...string) ([]*Record, error) {
	return nil, nil
}

func (n *noop) Write(rec ...*Record) error {
	return nil
}

func (n *noop) Delete(key ...string) error {
	return nil
}
