// Package model is an interface for structured data storage with schema awareness.
package model

import (
	"context"
	"errors"
)

var (
	// ErrNotFound is returned when a record doesn't exist.
	ErrNotFound = errors.New("not found")
	// ErrDuplicateKey is returned when a record with the same key already exists.
	ErrDuplicateKey = errors.New("duplicate key")
	// ErrNotRegistered is returned when a table has not been registered.
	ErrNotRegistered = errors.New("table not registered")
	// DefaultModel is the default model.
	DefaultModel Model = NewModel()
)

// Model is a structured data storage interface.
type Model interface {
	// Init initializes the model.
	Init(...Option) error
	// Register registers a struct type as a table.
	Register(v interface{}, opts ...RegisterOption) error
	// Create inserts a new record. Returns ErrDuplicateKey if key exists.
	Create(ctx context.Context, v interface{}) error
	// Read retrieves a record by key into v. Returns ErrNotFound if missing.
	Read(ctx context.Context, key string, v interface{}) error
	// Update modifies an existing record. Returns ErrNotFound if missing.
	Update(ctx context.Context, v interface{}) error
	// Delete removes a record by key. v is a pointer to the struct type.
	Delete(ctx context.Context, key string, v interface{}) error
	// List retrieves records matching the query. result must be a pointer to a slice of struct pointers.
	List(ctx context.Context, result interface{}, opts ...QueryOption) error
	// Count returns the number of matching records. v is a pointer to the struct type.
	Count(ctx context.Context, v interface{}, opts ...QueryOption) (int64, error)
	// Close closes the model.
	Close() error
	// String returns the name of the implementation.
	String() string
}

type Option func(*Options)

type RegisterOption func(*Schema)

// NewModel returns the default in-memory model.
func NewModel(opts ...Option) Model {
	return newMemoryModel(opts...)
}

// Register registers a struct type with the default model.
func Register(v interface{}, opts ...RegisterOption) error {
	return DefaultModel.Register(v, opts...)
}

// Create inserts a new record using the default model.
func Create(ctx context.Context, v interface{}) error {
	return DefaultModel.Create(ctx, v)
}

// Read retrieves a record by key using the default model.
func Read(ctx context.Context, key string, v interface{}) error {
	return DefaultModel.Read(ctx, key, v)
}

// Update modifies an existing record using the default model.
func Update(ctx context.Context, v interface{}) error {
	return DefaultModel.Update(ctx, v)
}

// Delete removes a record by key using the default model.
func Delete(ctx context.Context, key string, v interface{}) error {
	return DefaultModel.Delete(ctx, key, v)
}

// List retrieves records matching the query using the default model.
func List(ctx context.Context, result interface{}, opts ...QueryOption) error {
	return DefaultModel.List(ctx, result, opts...)
}

// Count returns the number of matching records using the default model.
func Count(ctx context.Context, v interface{}, opts ...QueryOption) (int64, error) {
	return DefaultModel.Count(ctx, v, opts...)
}
