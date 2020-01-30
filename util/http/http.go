package http

import (
	"net/http"

	"github.com/micro/go-micro/v2/client/selector"
	"github.com/micro/go-micro/v2/registry"
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
