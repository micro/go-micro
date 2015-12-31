package registry

import (
	"time"
)

type Options struct {
	Timeout time.Duration

	// Other options to be used by registry implementations
	Options map[string]string
}

func Timeout(t time.Duration) Option {
	return func(o *Options) {
		o.Timeout = t
	}
}
