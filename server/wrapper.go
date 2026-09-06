package server

import (
	"context"
)

// HandlerFunc represents a single method of a handler. It's used primarily
// for the wrappers. What's handed to the actual method is the concrete
// request and response types.
type HandlerFunc func(ctx context.Context, req Request, rsp interface{}) error

// SubscriberFunc represents a single method of a subscriber. It's used primarily
// for the wrappers. What's handed to the actual method is the concrete
// publication message.
type SubscriberFunc func(ctx context.Context, msg Message) error

// HandlerWrapper wraps the HandlerFunc and returns the equivalent.
type HandlerWrapper func(HandlerFunc) HandlerFunc

// SubscriberWrapper wraps the SubscriberFunc and returns the equivalent.
type SubscriberWrapper func(SubscriberFunc) SubscriberFunc

// StreamWrapper wraps a Stream interface and returns the equivalent.
// Because streams exist for the lifetime of a method invocation this
// is a convenient way to wrap a Stream as its in use for trace, monitoring,
// metrics, etc.
type StreamWrapper func(Stream) Stream

// BeforeHandler returns a HandlerWrapper that runs fn before the handler.
// If fn returns an error the handler is not invoked and the error is
// returned to the caller.
func BeforeHandler(fn func(context.Context, Request) error) HandlerWrapper {
	return func(h HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req Request, rsp interface{}) error {
			if err := fn(ctx, req); err != nil {
				return err
			}
			return h(ctx, req, rsp)
		}
	}
}

// AfterHandler returns a HandlerWrapper that runs fn after the handler,
// regardless of whether the handler returned an error. A handler error takes
// precedence over fn's error.
func AfterHandler(fn func(context.Context, Request) error) HandlerWrapper {
	return func(h HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req Request, rsp interface{}) error {
			herr := h(ctx, req, rsp)
			if aerr := fn(ctx, req); herr == nil {
				return aerr
			}
			return herr
		}
	}
}
