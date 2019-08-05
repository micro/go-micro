package http

import (
	"net/http"

	"github.com/alexapps/go-micro/client/selector"
	"github.com/alexapps/go-micro/registry"
)

func NewRoundTripper(opts ...Option) http.RoundTripper {
	options := Options{
		Registry: registry.DefaultRegistry,
	}
	for _, o := range opts {
		o(&options)
	}

	return &roundTripper{
		rt:   http.DefaultTransport,
		st:   selector.Random,
		opts: options,
	}
}
