// Package cache is a caching interface
package cache

type Cache interface {
	// Initialise options
	Init(...Option) error
	// Get a value
	Get(key string) (interface{}, error)
	// Set a value
	Set(key string, val interface{}) error
	// Delete a value
	Delete(key string) error
	// Name of the implementation
	String() string
}

type Option struct{}
