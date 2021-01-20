// Package opentracing provides wrappers for OpenTracing
package opentracing

import (
	"fmt"

	"context"
	"strings"

	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/metadata"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/v3/server"
	opentracing "github.com/opentracing/opentracing-go"
	opentracinglog "github.com/opentracing/opentracing-go/log"
)

type otWrapper struct {
	ot opentracing.Tracer
	client.Client
}

// StartSpanFromContext returns a new span with the given operation name and options. If a span
// is found in the context, it will be used as the parent of the resulting span.
func StartSpanFromContext(ctx context.Context, tracer opentracing.Tracer, name string, opts ...opentracing.StartSpanOption) (context.Context, opentracing.Span, error) {
	md, ok := metadata.FromContext(ctx)
	if !ok {
		md = make(metadata.Metadata)
	}

	// Find parent span.
	// First try to get span within current service boundary.
	// If there doesn't exist, try to get it from go-micro metadata(which is cross boundary)
	if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
		opts = append(opts, opentracing.ChildOf(parentSpan.Context()))
	} else if spanCtx, err := tracer.Extract(opentracing.TextMap, opentracing.TextMapCarrier(md)); err == nil {
		opts = append(opts, opentracing.ChildOf(spanCtx))
	}

	// allocate new map with only one element
	nmd := make(metadata.Metadata, 1)

	sp := tracer.StartSpan(name, opts...)

	if err := sp.Tracer().Inject(sp.Context(), opentracing.TextMap, opentracing.TextMapCarrier(nmd)); err != nil {
		return nil, nil, err
	}

	for k, v := range nmd {
		md.Set(strings.Title(k), v)
	}

	ctx = opentracing.ContextWithSpan(ctx, sp)
	ctx = metadata.NewContext(ctx, md)
	return ctx, sp, nil
}

func (o *otWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	name := fmt.Sprintf("%s.%s", req.Service(), req.Endpoint())
	ctx, span, err := StartSpanFromContext(ctx, o.ot, name)
	if err != nil {
		return err
	}
	defer span.Finish()
	if err = o.Client.Call(ctx, req, rsp, opts...); err != nil {
		span.LogFields(opentracinglog.String("error", err.Error()))
		span.SetTag("error", true)
	}
	return err
}

func (o *otWrapper) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	name := fmt.Sprintf("%s.%s", req.Service(), req.Endpoint())
	ctx, span, err := StartSpanFromContext(ctx, o.ot, name)
	if err != nil {
		return nil, err
	}
	defer span.Finish()
	stream, err := o.Client.Stream(ctx, req, opts...)
	if err != nil {
		span.LogFields(opentracinglog.String("error", err.Error()))
		span.SetTag("error", true)
	}
	return stream, err
}

func (o *otWrapper) Publish(ctx context.Context, p client.Message, opts ...client.PublishOption) error {
	name := fmt.Sprintf("Pub to %s", p.Topic())
	ctx, span, err := StartSpanFromContext(ctx, o.ot, name)
	if err != nil {
		return err
	}
	defer span.Finish()
	if err = o.Client.Publish(ctx, p, opts...); err != nil {
		span.LogFields(opentracinglog.String("error", err.Error()))
		span.SetTag("error", true)
	}
	return err
}

// NewClientWrapper accepts an open tracing Trace and returns a Client Wrapper
func NewClientWrapper(ot opentracing.Tracer) client.Wrapper {
	return func(c client.Client) client.Client {
		if ot == nil {
			ot = opentracing.GlobalTracer()
		}
		return &otWrapper{ot, c}
	}
}

// NewCallWrapper accepts an opentracing Tracer and returns a Call Wrapper
func NewCallWrapper(ot opentracing.Tracer) client.CallWrapper {
	return func(cf client.CallFunc) client.CallFunc {
		return func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
			if ot == nil {
				ot = opentracing.GlobalTracer()
			}
			name := fmt.Sprintf("%s.%s", req.Service(), req.Endpoint())
			ctx, span, err := StartSpanFromContext(ctx, ot, name)
			if err != nil {
				return err
			}
			defer span.Finish()
			if err = cf(ctx, node, req, rsp, opts); err != nil {
				span.LogFields(opentracinglog.String("error", err.Error()))
				span.SetTag("error", true)
			}
			return err
		}
	}
}

// NewHandlerWrapper accepts an opentracing Tracer and returns a Handler Wrapper
func NewHandlerWrapper(ot opentracing.Tracer) server.HandlerWrapper {
	return func(h server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			if ot == nil {
				ot = opentracing.GlobalTracer()
			}
			name := fmt.Sprintf("%s.%s", req.Service(), req.Endpoint())
			ctx, span, err := StartSpanFromContext(ctx, ot, name)
			if err != nil {
				return err
			}
			defer span.Finish()
			if err = h(ctx, req, rsp); err != nil {
				span.LogFields(opentracinglog.String("error", err.Error()))
				span.SetTag("error", true)
			}
			return err
		}
	}
}

// NewSubscriberWrapper accepts an opentracing Tracer and returns a Subscriber Wrapper
func NewSubscriberWrapper(ot opentracing.Tracer) server.SubscriberWrapper {
	return func(next server.SubscriberFunc) server.SubscriberFunc {
		return func(ctx context.Context, msg server.Message) error {
			name := "Sub from " + msg.Topic()
			if ot == nil {
				ot = opentracing.GlobalTracer()
			}
			ctx, span, err := StartSpanFromContext(ctx, ot, name)
			if err != nil {
				return err
			}
			defer span.Finish()
			if err = next(ctx, msg); err != nil {
				span.LogFields(opentracinglog.String("error", err.Error()))
				span.SetTag("error", true)
			}
			return err
		}
	}
}
