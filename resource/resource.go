// Package resource is an external resource
package resource

// Resource represents an external resource
// which can be run alongside a service e.g
// a database.
type Resource interface {
	// Initialise
	Init(...Option) error
	// Get options
	Options() Options
	// Name of the resource
	Name() string
	// Type of resource
	Type() string
	// Run the resource
	Run() error
	// Resource implementation
	String()
}

type Options struct{}

type Option func(*Options) error
