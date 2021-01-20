package datadog

import (
	"context"

	log "github.com/asim/go-micro/v3/logger"
	"github.com/asim/go-micro/v3/metadata"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// StartSpanFromContext returns a new span with the given operation name and options. If a span
// is found in the context, it will be used as the parent of the resulting span.
func StartSpanFromContext(ctx context.Context, operationName string, opts ...tracer.StartSpanOption) (tracer.Span, context.Context) {
	md, ok := metadata.FromContext(ctx)
	if !ok {
		md = make(map[string]string)
	}

	if spanCtx, err := tracer.Extract(tracer.TextMapCarrier(md)); err == nil {
		opts = append(opts, tracer.ChildOf(spanCtx))
	}

	span, ctx := tracer.StartSpanFromContext(ctx, operationName, opts...)

	if err := tracer.Inject(span.Context(), tracer.TextMapCarrier(md)); err != nil {
		log.Errorf("error while injecting trace to context: %s\n", err)
	}

	return span, metadata.NewContext(ctx, md)
}
