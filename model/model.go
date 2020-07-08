// Package model is an interface for data modelling
package model

import (
	"github.com/micro/go-micro/v2/store"
	"github.com/micro/go-micro/v2/sync"
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
	// The value associated with the entity
	Value() interface{}
	// Attributes of the enity
	Attributes() map[string]interface{}
}

type Options struct {
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
