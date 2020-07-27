// Package model is an interface for data modelling
package model

import (
	"github.com/micro/go-micro/v3/codec"
	"github.com/micro/go-micro/v3/store"
	"github.com/micro/go-micro/v3/sync"
)

// Model provides an interface for data modelling
type Model interface {
	// Initialise options
	Init(...Option) error
	// NewEntity creates a new entity to store or access
	NewEntity(name string, value interface{}) Entity
	// Create a value
	Create(Entity) error
	// Read values
	Read(...ReadOption) ([]Entity, error)
	// Update the value
	Update(Entity) error
	// Delete an entity
	Delete(...DeleteOption) error
	// Implementation of the model
	String() string
}

type Entity interface {
	// Unique id of the entity
	Id() string
	// Name of the entity
	Name() string
	// The value associated with the entity
	Value() interface{}
	// Attributes of the entity
	Attributes() map[string]interface{}
	// Read a value as a concrete type
	Read(v interface{}) error
}

type Options struct {
	// Database to write to
	Database string
	// for serialising
	Codec codec.Marshaler
	// for locking
	Sync sync.Sync
	// for storage
	Store store.Store
}

type Option func(o *Options)

type ReadOptions struct{}

type ReadOption func(o *ReadOptions)

type DeleteOptions struct{}

type DeleteOption func(o *DeleteOptions)
