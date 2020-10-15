// Package cache is a caching interface
package cache

// Cache is an interface for caching
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

type Options struct {
	Nodes []string
}

type Option func(o *Options)

// Nodes sets the nodes for the cache
func Nodes(v ...string) Option {
	return func(o *Options) {
		o.Nodes = v
	}
}
