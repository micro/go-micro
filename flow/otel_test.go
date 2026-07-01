package flow

import (
	"context"
	"testing"

	"go-micro.dev/v6/ai"
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
	ctx := withTestRunInfo(context.Background(), "agent-run-otel")
	if err := f.Execute(ctx, "start"); err != nil {
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
			if attrs[AttrFlowName] != "observed" || attrs[AttrFlowStatus] != "done" || attrs[AttrFlowParentID] != "agent-run-otel" {
				t.Fatalf("run span attributes = %#v", attrs)
			}
		case spanNameFlowStep:
			seen[spanNameFlowStep] = true
			if attrs[AttrFlowName] != "observed" || attrs[AttrFlowStepName] != "inspect" || attrs[AttrFlowParentID] != "agent-run-otel" {
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

func withTestRunInfo(ctx context.Context, runID string) context.Context {
	return ai.WithRunInfo(ctx, ai.RunInfo{RunID: runID, Agent: "planner"})
}

func TestScheduledFlowOpenTelemetryDispatchAttributes(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exp))

	step := Step{Name: "summarize", Run: func(ctx context.Context, in State) (State, error) {
		in.Data = []byte("queued")
		return in, nil
	}}
	f := New("scheduled-observed", Trigger("schedule.daily"), WithCheckpoint(StoreCheckpoint(store.NewMemoryStore(), "scheduled-observed")), TraceProvider(tp), Steps(step))
	if err := Scheduled(f, "daily ops review").Tick(context.Background()); err != nil {
		t.Fatal(err)
	}

	for _, span := range exp.GetSpans().Snapshots() {
		if span.Name() != spanNameFlowRun {
			continue
		}
		attrs := flowSpanAttributes(span.Attributes())
		if attrs[AttrFlowDispatch] != "schedule" || attrs[AttrFlowTrigger] != "schedule.daily" {
			t.Fatalf("scheduled run span dispatch attributes = %#v", attrs)
		}
		return
	}
	t.Fatal("flow run span not emitted")
}
