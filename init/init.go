// Package init is an interface for initialising options
package init

// Init is used for initialisation
type Init interface {
	// Initialise options
	Init(...Option) error
	// Options returns the current options
	Options() *Options
	// Value returns an option value
	Value(k interface{}) (interface{}, bool)
	// The name for who these options exist
	String() string
}

// NewInit returns a new initialiser
func NewInit(opts ...Option) Init {
	i := new(defaultInit)
	i.Init(opts...)
	return i
}
