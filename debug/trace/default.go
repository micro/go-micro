package trace

import (
	"context"
	"time"

	"github.com/asim/go-micro/v3/util/ring"
	"github.com/google/uuid"
)

type memTracer struct {
	opts Options

	// ring buffer of traces
	buffer *ring.Buffer
}

func (t *memTracer) Read(opts ...ReadOption) ([]*Span, error) {
	var options ReadOptions
	for _, o := range opts {
		o(&options)
	}

	sp := t.buffer.Get(t.buffer.Size())

	spans := make([]*Span, 0, len(sp))

	for _, span := range sp {
		val := span.Value.(*Span)
		// skip if trace id is specified and doesn't match
		if len(options.Trace) > 0 && val.Trace != options.Trace {
			continue
		}
		spans = append(spans, val)
	}

	return spans, nil
}

func (t *memTracer) Start(ctx context.Context, name string) (context.Context, *Span) {
	span := &Span{
		Name:     name,
		Trace:    uuid.New().String(),
		Id:       uuid.New().String(),
		Started:  time.Now(),
		Metadata: make(map[string]string),
	}

	// return span if no context
	if ctx == nil {
		return ToContext(context.Background(), span.Trace, span.Id), span
	}
	traceID, parentSpanID, ok := FromContext(ctx)
	// If the trace can not be found in the header,
	// that means this is where the trace is created.
	if !ok {
		return ToContext(ctx, span.Trace, span.Id), span
	}

	// set trace id
	span.Trace = traceID
	// set parent
	span.Parent = parentSpanID

	// return the span
	return ToContext(ctx, span.Trace, span.Id), span
}

func (t *memTracer) Finish(s *Span) error {
	// set finished time
	s.Duration = time.Since(s.Started)
	// save the span
	t.buffer.Put(s)

	return nil
}

func NewTracer(opts ...Option) Tracer {
	var options Options
	for _, o := range opts {
		o(&options)
	}

	return &memTracer{
		opts: options,
		// the last 256 requests
		buffer: ring.New(256),
	}
}
