package agent

import (
	"context"
	"testing"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/store"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type otelTestModel struct{ opts ai.Options }

func (m *otelTestModel) Init(opts ...ai.Option) error {
	for _, o := range opts {
		o(&m.opts)
	}
	return nil
}
func (m *otelTestModel) Options() ai.Options { return m.opts }
func (m *otelTestModel) String() string      { return "oteltest" }
func (m *otelTestModel) Stream(context.Context, *ai.Request, ...ai.GenerateOption) (ai.Stream, error) {
	return nil, nil
}
func (m *otelTestModel) Generate(ctx context.Context, req *ai.Request, opts ...ai.GenerateOption) (*ai.Response, error) {
	if m.opts.ToolHandler != nil {
		_ = m.opts.ToolHandler(ctx, ai.ToolCall{ID: "call-1", Name: "probe", Input: map[string]any{"ok": true}})
	}
	return &ai.Response{Reply: "done", Usage: ai.Usage{InputTokens: 2, OutputTokens: 3, TotalTokens: 5}}, nil
}

func init() {
	ai.Register("oteltest", func(opts ...ai.Option) ai.Model { return &otelTestModel{opts: ai.NewOptions(opts...)} })
}

func TestAgentOpenTelemetrySpans(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exp))
	st := store.NewMemoryStore()
	a := New(Name("runner"), Provider("oteltest"), Model("unit-model"), WithStore(st), TraceProvider(tp), WithTool("probe", "probe", nil, func(context.Context, map[string]any) (string, error) { return "ok", nil }))
	if _, err := a.Ask(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	spans := exp.GetSpans().Snapshots()
	want := map[string]bool{spanNameRun: false, spanNameModelCall: false, spanNameToolCall: false}
	for _, s := range spans {
		if _, ok := want[s.Name()]; ok {
			want[s.Name()] = true
		}
	}
	for name, seen := range want {
		if !seen {
			t.Fatalf("span %s not emitted; got %d spans", name, len(spans))
		}
	}
	keys, err := store.Scope(st, "agent", "runner").List(store.ListPrefix("runs/"))
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) == 0 {
		t.Fatal("expected run events to be recorded")
	}
}

func TestAgentOpenTelemetryNoopWhenUnconfigured(t *testing.T) {
	st := store.NewMemoryStore()
	a := New(Name("runner-noop"), Provider("oteltest"), WithStore(st), WithTool("probe", "probe", nil, func(context.Context, map[string]any) (string, error) { return "ok", nil }))
	if _, err := a.Ask(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	keys, err := store.Scope(st, "agent", "runner-noop").List(store.ListPrefix("runs/"))
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected no run timeline without TraceProvider, got %v", keys)
	}
	if _, ok := a.(*agentImpl).model.(*tracedModel); ok {
		t.Fatal("model should not be wrapped when TraceProvider is nil")
	}
}
