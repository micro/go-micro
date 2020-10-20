package http

import "github.com/asim/go-micro/v3/router"

type Options struct {
	Router router.Router
}

type Option func(*Options)

func WithRouter(r router.Router) Option {
	return func(o *Options) {
		o.Router = r
	}
}
