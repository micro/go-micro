package http

import (
	"context"
	"net/http"

	"github.com/micro/go-micro/v2/transport"
)

// Handle registers the handler for the given pattern.
func Handle(pattern string, handler http.Handler) transport.Option {
	return func(o *transport.Options) {
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
