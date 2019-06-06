package init

import (
	"sync"
)

// Options holds the set of option values and protects them
type Options struct {
	sync.RWMutex
	values map[interface{}]interface{}
}

// Option gives access to options
type Option func(o *Options) error

// Get a value from options
func (o *Options) Value(k interface{}) (interface{}, bool) {
	o.RLock()
	defer o.RUnlock()
	v, ok := o.values[k]
	return v, ok
}

// Set a value in the options
func (o *Options) SetValue(k, v interface{}) error {
	o.Lock()
	defer o.Unlock()
	if o.values == nil {
		o.values = map[interface{}]interface{}{}
	}
	o.values[k] = v
	return nil
}

// SetOption executes an option
func (o *Options) SetOption(op Option) error {
	return op(o)
}

// WithValue allows you to set any value within the options
func WithValue(k, v interface{}) Option {
	return func(o *Options) error {
		return o.SetValue(k, v)
	}
}

// WithOption gives you the ability to create an option that accesses values
func WithOption(o Option) Option {
	return o
}

// String sets the string
func String(s string) Option {
	return WithValue(stringKey{}, s)
}
