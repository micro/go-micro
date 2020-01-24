package trace

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/util/ring"
)

type trace struct {
	opts Options

	// ring buffer of traces
	buffer *ring.Buffer
}

func (t *trace) Read(opts ...ReadOption) ([]*Span, error) {
	var options ReadOptions
	for _, o := range opts {
		o(&options)
	}

	sp := t.buffer.Get(t.buffer.Size())

	var spans []*Span

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

func (t *trace) Start(ctx context.Context, name string) (context.Context, *Span) {
	span := &Span{
		Name:     name,
		Trace:    uuid.New().String(),
		Id:       uuid.New().String(),
		Started:  time.Now(),
		Metadata: make(map[string]string),
	}

	// return span if no context
	if ctx == nil {
		return context.Background(), span
	}

	s, ok := FromContext(ctx)
	if !ok {
		return ctx, span
	}

	// set trace id
	span.Trace = s.Trace
	// set parent
	span.Parent = s.Id

	// return the sapn
	return ctx, span
}

func (t *trace) Finish(s *Span) error {
	// set finished time
	s.Duration = time.Since(s.Started)

	// save the span
	t.buffer.Put(s)

	return nil
}

func NewTrace(opts ...Option) Trace {
	var options Options
	for _, o := range opts {
		o(&options)
	}

	return &trace{
		opts: options,
		// the last 64 requests
		buffer: ring.New(64),
	}
}
