// Package cache provides a caching interface
package cache

type Cache interface {
	// Get returns a val deserialised into it
	Get(key string, val interface{}) error
	// Set a value
	Set(key string, val interface{}) error
	// Delete a value
	Del(key string) error
	// List keys
	Keys() ([]string, error)
}
