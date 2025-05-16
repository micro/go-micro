// Package store is an interface for distributed data storage.
// The design document is located at https://github.com/micro/development/blob/master/design/store.md
package store

import (
	"errors"
	"time"

	"encoding/json"
)

var (
	// ErrNotFound is returned when a key doesn't exist.
	ErrNotFound = errors.New("not found")
	// DefaultStore is the memory store.
	DefaultStore Store = NewStore()
)

// Store is a data storage interface.
type Store interface {
	// Init initializes the store. It must perform any required setup on the backing storage implementation and check that it is ready for use, returning any errors.
	Init(...Option) error
	// Options allows you to view the current options.
	Options() Options
	// Read takes a single key name and optional ReadOptions. It returns matching []*Record or an error.
	Read(key string, opts ...ReadOption) ([]*Record, error)
	// Write() writes a record to the store, and returns an error if the record was not written.
	Write(r *Record, opts ...WriteOption) error
	// Delete removes the record with the corresponding key from the store.
	Delete(key string, opts ...DeleteOption) error
	// List returns any keys that match, or an empty list with no error if none matched.
	List(opts ...ListOption) ([]string, error)
	// Close the store
	Close() error
	// String returns the name of the implementation.
	String() string
}

// Record is an item stored or retrieved from a Store.
type Record struct {
	// Any associated metadata for indexing
	Metadata map[string]interface{} `json:"metadata"`
	// The key to store the record
	Key string `json:"key"`
	// The value within the record
	Value []byte `json:"value"`
	// Time to expire a record: TODO: change to timestamp
	Expiry time.Duration `json:"expiry,omitempty"`
}

func NewStore(opts ...Option) Store {
	return NewFileStore(opts...)
}

func NewRecord(key string, val interface{}) *Record {
	b, _ := json.Marshal(val)
	return &Record{
		Key:   key,
		Value: b,
	}
}

// Encode will marshal any type into the byte Value field
func (r *Record) Encode(v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	r.Value = b
	return nil
}

// Decode is a convenience helper for decoding records
func (r *Record) Decode(v interface{}) error {
	return json.Unmarshal(r.Value, v)
}

// Read records
func Read(key string, opts ...ReadOption) ([]*Record, error) {
	// execute the query
	return DefaultStore.Read(key, opts...)
}

// Write a record to the store
func Write(r *Record) error {
	return DefaultStore.Write(r)
}

// Delete removes the record with the corresponding key from the store.
func Delete(key string) error {
	return DefaultStore.Delete(key)
}

// List returns any keys that match, or an empty list with no error if none matched.
func List(opts ...ListOption) ([]string, error) {
	return DefaultStore.List(opts...)
}
