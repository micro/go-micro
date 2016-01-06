package registry

import (
	"time"

	"golang.org/x/net/context"
)

type Options struct {
	Timeout time.Duration

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

func Timeout(t time.Duration) Option {
	return func(o *Options) {
		o.Timeout = t
	}
}
