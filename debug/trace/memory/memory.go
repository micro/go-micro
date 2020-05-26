package memory

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/v2/debug/trace"
	"github.com/micro/go-micro/v2/util/ring"
)

type Tracer struct {
	opts trace.Options

	// ring buffer of traces
	buffer *ring.Buffer
}

func (t *Tracer) Read(opts ...trace.ReadOption) ([]*trace.Span, error) {
	var options trace.ReadOptions
	for _, o := range opts {
		o(&options)
	}

	sp := t.buffer.Get(t.buffer.Size())

	spans := make([]*trace.Span, 0, len(sp))

	for _, span := range sp {
		val := span.Value.(*trace.Span)
		// skip if trace id is specified and doesn't match
		if len(options.Trace) > 0 && val.Trace != options.Trace {
			continue
		}
		spans = append(spans, val)
	}

	return spans, nil
}

func (t *Tracer) Start(ctx context.Context, name string) (context.Context, *trace.Span) {
	span := &trace.Span{
		Name:     name,
		Trace:    uuid.New().String(),
		Id:       uuid.New().String(),
		Started:  time.Now(),
		Metadata: make(map[string]string),
	}

	// return span if no context
	if ctx == nil {
		return trace.ToContext(context.Background(), span.Trace, span.Id), span
	}
	traceID, parentSpanID, ok := trace.FromContext(ctx)
	// If the trace can not be found in the header,
	// that means this is where the trace is created.
	if !ok {
		return trace.ToContext(ctx, span.Trace, span.Id), span
	}

	// set trace id
	span.Trace = traceID
	// set parent
	span.Parent = parentSpanID

	// return the span
	return trace.ToContext(ctx, span.Trace, span.Id), span
}

func (t *Tracer) Finish(s *trace.Span) error {
	// set finished time
	s.Duration = time.Since(s.Started)
	// save the span
	t.buffer.Put(s)

	return nil
}

func NewTracer(opts ...trace.Option) trace.Tracer {
	var options trace.Options
	for _, o := range opts {
		o(&options)
	}

	return &Tracer{
		opts: options,
		// the last 256 requests
		buffer: ring.New(256),
	}
}
