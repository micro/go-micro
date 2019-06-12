// Package options provides a way to initialise options
package options

import (
	"sync"
)

// Options is used for initialisation
type Options interface {
	// Initialise options
	Init(...Option) error
	// Options returns the current options
	Values() *Values
	// The name for who these options exist
	String() string
}

// Values holds the set of option values and protects them
type Values struct {
	sync.RWMutex
	values map[interface{}]interface{}
}

// Option gives access to options
type Option func(o *Values) error

// Get a value from options
func (o *Values) Get(k interface{}) (interface{}, bool) {
	o.RLock()
	defer o.RUnlock()
	v, ok := o.values[k]
	return v, ok
}

// Set a value in the options
func (o *Values) Set(k, v interface{}) error {
	o.Lock()
	defer o.Unlock()
	if o.values == nil {
		o.values = map[interface{}]interface{}{}
	}
	o.values[k] = v
	return nil
}

// SetOption executes an option
func (o *Values) Option(op Option) error {
	return op(o)
}

// WithValue allows you to set any value within the options
func WithValue(k, v interface{}) Option {
	return func(o *Values) error {
		return o.Set(k, v)
	}
}

// WithOption gives you the ability to create an option that accesses values
func WithOption(o Option) Option {
	return o
}

// String sets the string
func WithString(s string) Option {
	return WithValue(stringKey{}, s)
}

// NewOptions returns a new initialiser
func NewOptions(opts ...Option) Options {
	o := new(defaultOptions)
	o.Init(opts...)
	return o
}
