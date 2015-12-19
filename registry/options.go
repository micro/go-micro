package registry

import (
	"time"
)

type Options struct {
	Timeout time.Duration
}

func Timeout(t time.Duration) Option {
	return func(o *Options) {
		o.Timeout = t
	}
}
