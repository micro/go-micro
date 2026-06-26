package flow

import (
	"context"
	"testing"

	"go-micro.dev/v6/store"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestFlowOpenTelemetrySpans(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exp))

	step := Step{Name: "inspect", Run: func(ctx context.Context, in State) (State, error) {
		in.Data = []byte("done")
		return in, nil
	}}
	f := New("observed", WithCheckpoint(StoreCheckpoint(store.NewMemoryStore(), "observed")), TraceProvider(tp), Steps(step))
	if err := f.Execute(context.Background(), "start"); err != nil {
		t.Fatal(err)
	}

	spans := exp.GetSpans().Snapshots()
	seen := map[string]bool{spanNameFlowRun: false, spanNameFlowStep: false}
	var runID string
	for _, span := range spans {
		attrs := flowSpanAttributes(span.Attributes())
		switch span.Name() {
		case spanNameFlowRun:
			seen[spanNameFlowRun] = true
			runID = attrs[AttrFlowRunID]
			if attrs[AttrFlowName] != "observed" || attrs[AttrFlowStatus] != "done" {
				t.Fatalf("run span attributes = %#v", attrs)
			}
		case spanNameFlowStep:
			seen[spanNameFlowStep] = true
			if attrs[AttrFlowName] != "observed" || attrs[AttrFlowStepName] != "inspect" {
				t.Fatalf("step span attributes = %#v", attrs)
			}
		}
	}
	for name, ok := range seen {
		if !ok {
			t.Fatalf("span %s not emitted; got %d spans", name, len(spans))
		}
	}
	if runID == "" {
		t.Fatal("run span missing run id")
	}
	for _, span := range spans {
		if span.Name() != spanNameFlowStep {
			continue
		}
		attrs := flowSpanAttributes(span.Attributes())
		if attrs[AttrFlowRunID] != runID {
			t.Fatalf("step span run id = %q, want %q", attrs[AttrFlowRunID], runID)
		}
	}
}

func flowSpanAttributes(attrs []attribute.KeyValue) map[string]string {
	out := make(map[string]string, len(attrs))
	for _, attr := range attrs {
		out[string(attr.Key)] = attr.Value.AsString()
	}
	return out
}
