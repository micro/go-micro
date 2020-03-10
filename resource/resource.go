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
	// Run the resource
	Run() error
	// Name of the resource
	String()
}

type Options struct{}

type Option func(*Options) error
