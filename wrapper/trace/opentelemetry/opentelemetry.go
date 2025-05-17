package opentelemetry

import (
	"context"
	"strings"

	"go-micro.dev/v5/metadata"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	instrumentationName = "github.com/micro/plugins/v5/wrapper/trace/opentelemetry"
)

// StartSpanFromContext returns a new span with the given operation name and options. If a span
// is found in the context, it will be used as the parent of the resulting span.
func StartSpanFromContext(ctx context.Context, tp trace.TracerProvider, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	md, ok := metadata.FromContext(ctx)
	if !ok {
		md = make(metadata.Metadata)
	}
	propagator, carrier := otel.GetTextMapPropagator(), make(propagation.MapCarrier)
	for k, v := range md {
		for _, f := range propagator.Fields() {
			if strings.EqualFold(k, f) {
				carrier[f] = v
			}
		}
	}
	ctx = propagator.Extract(ctx, carrier)
	spanCtx := trace.SpanContextFromContext(ctx)
	ctx = baggage.ContextWithBaggage(ctx, baggage.FromContext(ctx))

	var tracer trace.Tracer
	var span trace.Span
	if tp != nil {
		tracer = tp.Tracer(instrumentationName)
	} else {
		tracer = otel.Tracer(instrumentationName)
	}
	ctx, span = tracer.Start(trace.ContextWithRemoteSpanContext(ctx, spanCtx), name, opts...)

	carrier = make(propagation.MapCarrier)
	propagator.Inject(ctx, carrier)
	for k, v := range carrier {
		//lint:ignore SA1019 no unicode punctution handle needed
		md.Set(strings.Title(k), v)
	}
	ctx = metadata.NewContext(ctx, md)

	return ctx, span
}
