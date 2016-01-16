package registry

import (
	"time"

	"golang.org/x/net/context"
)

type Options struct {
	Timeout time.Duration
	Secure  bool

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

func Timeout(t time.Duration) Option {
	return func(o *Options) {
		o.Timeout = t
	}
}

// Secure communication with the registry
func Secure(b bool) Option {
	return func(o *Options) {
		o.Secure = b
	}
}
