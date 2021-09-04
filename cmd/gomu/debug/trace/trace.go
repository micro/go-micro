package trace

import (
	"context"
	"runtime"

	"github.com/opentracing/opentracing-go"
)

// NewSpan accepts a context and returns an OpenTracing span. Can be used to
// nest spans.
func NewSpan(ctx context.Context) opentracing.Span {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	span := opentracing.StartSpan(
		runtime.FuncForPC(pc[0]).Name(),
		opentracing.ChildOf(opentracing.SpanFromContext(ctx).Context()),
	)
	return span
}
