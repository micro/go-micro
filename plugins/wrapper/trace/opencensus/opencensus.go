// Package opencensus provides wrappers for OpenCensus tracing.
package opencensus

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/micro/go-micro/v2/client"
	log "github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/metadata"
	"github.com/micro/go-micro/v2/server"
	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
)

const (
	// TracePropagationField is the key for the tracing context
	// that will be injected in go-micro's metadata.
	TracePropagationField = "X-Trace-Context"
)

// clientWrapper wraps an RPC client and adds tracing.
type clientWrapper struct {
	client.Client
}

func injectTraceIntoCtx(ctx context.Context, span *trace.Span) context.Context {
	spanCtx := propagation.Binary(span.SpanContext())
	metadata.Set(ctx, TracePropagationField, base64.RawStdEncoding.EncodeToString(spanCtx))
	return ctx
}

// Call implements client.Client.Call.
func (w *clientWrapper) Call(
	ctx context.Context,
	req client.Request,
	rsp interface{},
	opts ...client.CallOption) (err error) {
	t := newRequestTracker(req, ClientProfile)
	ctx = t.start(ctx, true)

	defer func() { t.end(ctx, err) }()

	ctx = injectTraceIntoCtx(ctx, t.span)

	err = w.Client.Call(ctx, req, rsp, opts...)
	return
}

// Publish implements client.Client.Publish.
func (w *clientWrapper) Publish(ctx context.Context, p client.Message, opts ...client.PublishOption) (err error) {
	t := newEventTracker(p, ClientProfile)
	ctx = t.start(ctx, true)

	defer func() { t.end(ctx, err) }()

	ctx = injectTraceIntoCtx(ctx, t.span)

	err = w.Client.Publish(ctx, p, opts...)
	return
}

// NewClientWrapper returns a client.Wrapper
// that adds monitoring to outgoing requests.
func NewClientWrapper() client.Wrapper {
	return func(c client.Client) client.Client {
		return &clientWrapper{c}
	}
}

func getTraceFromCtx(ctx context.Context) *trace.SpanContext {
	encodedTraceCtx, ok := metadata.Get(ctx, TracePropagationField)
	if !ok {
		return nil
	}

	traceCtxBytes, err := base64.RawStdEncoding.DecodeString(encodedTraceCtx)
	if err != nil {
		log.Errorf("Could not decode trace context: %s", err.Error())
		return nil
	}

	spanCtx, ok := propagation.FromBinary(traceCtxBytes)
	if !ok {
		log.Errorf("Could not decode trace context from binary")
		return nil
	}

	return &spanCtx
}

// NewHandlerWrapper returns a server.HandlerWrapper
// that adds tracing to incoming requests.
func NewHandlerWrapper() server.HandlerWrapper {
	return func(fn server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) (err error) {
			t := newRequestTracker(req, ServerProfile)
			ctx = t.start(ctx, false)

			defer func() { t.end(ctx, err) }()

			spanCtx := getTraceFromCtx(ctx)
			if spanCtx != nil {
				ctx, t.span = trace.StartSpanWithRemoteParent(
					ctx,
					fmt.Sprintf("rpc/%s/%s/%s", ServerProfile.Role, req.Service(), req.Endpoint()),
					*spanCtx,
				)
			} else {
				ctx, t.span = trace.StartSpan(
					ctx,
					fmt.Sprintf("rpc/%s/%s/%s", ServerProfile.Role, req.Service(), req.Endpoint()),
				)
			}

			err = fn(ctx, req, rsp)
			return
		}
	}
}

// NewSubscriberWrapper returns a server.SubscriberWrapper
// that adds tracing to subscription requests.
func NewSubscriberWrapper() server.SubscriberWrapper {
	return func(fn server.SubscriberFunc) server.SubscriberFunc {
		return func(ctx context.Context, p server.Message) (err error) {
			t := newEventTracker(p, ServerProfile)
			ctx = t.start(ctx, false)

			defer func() { t.end(ctx, err) }()

			spanCtx := getTraceFromCtx(ctx)
			if spanCtx != nil {
				ctx, t.span = trace.StartSpanWithRemoteParent(
					ctx,
					fmt.Sprintf("rpc/%s/pubsub/%s", ServerProfile.Role, p.Topic()),
					*spanCtx,
				)
			} else {
				ctx, t.span = trace.StartSpan(
					ctx,
					fmt.Sprintf("rpc/%s/pubsub/%s", ServerProfile.Role, p.Topic()),
				)
			}

			err = fn(ctx, p)
			return
		}
	}
}
