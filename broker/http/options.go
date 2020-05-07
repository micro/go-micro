package http

import (
	"context"
	"net/http"

	"github.com/micro/go-micro/v2/broker"
)

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
