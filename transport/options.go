package transport

import (
	"time"

	"golang.org/x/net/context"
)

type Options struct {
	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type DialOptions struct {
	Stream  bool
	Timeout time.Duration

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

func WithStream() DialOption {
	return func(o *DialOptions) {
		o.Stream = true
	}
}

func WithTimeout(d time.Duration) DialOption {
	return func(o *DialOptions) {
		o.Timeout = d
	}
}
