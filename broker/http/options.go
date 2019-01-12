package http

import (
	"context"
	"net/http"

	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/registry"
)

type registryKey struct{}

// Handle registers the handler for the given pattern.
func Handle(pattern string, handler http.Handler) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		handlers, ok := o.Context.Value("http_handlers").(map[string]http.Handler)
		if !ok {
			handlers = make(map[string]http.Handler)
		}
		handlers[pattern] = handler
		o.Context = context.WithValue(o.Context, "http_handlers", handlers)
	}
}

// Registry sets the broker internal registry
func Registry(r registry.Registry) broker.Option {
	return func(o *broker.Options) {
		o.Context = context.WithValue(o.Context, registryKey{}, r)
	}
}
