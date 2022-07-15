package api

import (
	"go-micro.dev/v4/api/router"
)

func NewOptions(opts ...Option) Options {
	options := Options{
		Address: ":8080",
	}

	for _, o := range opts {
		o(&options)
	}

	return options
}

// WithRouter sets the router to use e.g static or registry
func WithRouter(r router.Router) Option {
	return func(o *Options) error {
		o.Router = r
		return nil
	}
}
