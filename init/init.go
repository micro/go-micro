// Package init is an interface for initialising options
package init

// Options is used for initialisation
type Options interface {
	// Initialise options
	Init(...Option) error
	// Options returns the current options
	Options() Options
	// Value returns an option value
	Value(k interface{}) (interface{}, bool)
	// The name for who these options exist
	String() string
}

// NewOptions returns a new initialiser
func NewOptions(opts ...Option) Options {
	o := new(defaultOptions)
	o.Init(opts...)
	return o
}
