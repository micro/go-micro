// Package datadog provides wrappers for Datadog ddtrace
package datadog

import (
	"github.com/micro/go-micro/v2/registry"

	"context"

	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/server"
)

var noDebugStack = true

// SetNoDebugStack ...
func SetNoDebugStack(val bool) {
	noDebugStack = val
}

type ddWrapper struct {
	client.Client
}

func (d *ddWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) (err error) {
	t := newRequestTracker(req, ClientProfile)
	ctx = t.StartSpanFromContext(ctx)

	defer func() {
		t.finishWithError(err, noDebugStack)
	}()

	err = d.Client.Call(ctx, req, rsp, opts...)
	return
}

func (d *ddWrapper) Publish(ctx context.Context, p client.Message, opts ...client.PublishOption) (err error) {
	t := newEventTracker(p, ClientProfile)
	ctx = t.StartSpanFromContext(ctx)

	defer func() {
		t.finishWithError(err, noDebugStack)
	}()

	err = d.Client.Publish(ctx, p, opts...)
	return
}

// NewClientWrapper returns a Client wrapped in tracer
func NewClientWrapper() client.Wrapper {
	return func(c client.Client) client.Client {
		return &ddWrapper{c}
	}
}

// NewCallWrapper returns a Call Wrapper
func NewCallWrapper() client.CallWrapper {
	return func(cf client.CallFunc) client.CallFunc {
		return func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
			t := newRequestTracker(req, ClientProfile)
			ctx = t.StartSpanFromContext(ctx)

			defer func() {
				t.finishWithError(nil, noDebugStack)
			}()

			return cf(ctx, node, req, rsp, opts)
		}
	}
}

// NewHandlerWrapper returns a Handler Wrapper
func NewHandlerWrapper() server.HandlerWrapper {
	return func(h server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) (err error) {
			if req.Endpoint() != "Debug.Health" {
				t := newRequestTracker(req, ServerProfile)
				ctx = t.StartSpanFromContext(ctx)
				defer func() {
					t.finishWithError(err, noDebugStack)
				}()
			}

			err = h(ctx, req, rsp)

			return
		}
	}
}

// NewSubscriberWrapper returns a Subscriber Wrapper
func NewSubscriberWrapper() server.SubscriberWrapper {
	return func(next server.SubscriberFunc) server.SubscriberFunc {
		return func(ctx context.Context, msg server.Message) (err error) {
			t := newEventTracker(msg, ServerProfile)
			ctx = t.StartSpanFromContext(ctx)
			defer func() {
				t.finishWithError(err, noDebugStack)
			}()

			err = next(ctx, msg)
			return
		}
	}
}
