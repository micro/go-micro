package agent

import (
	"context"
	"encoding/json"
	"testing"
	"time"

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

func TestLoadRunEventsSortsTimelineKeys(t *testing.T) {
	st := store.NewMemoryStore()
	scoped := store.Scope(st, "agent", "runner")
	runID := "run-1"
	events := []RunEvent{
		{Time: time.Unix(0, 3), RunID: runID, Agent: "runner", Kind: "tool", Name: "third"},
		{Time: time.Unix(0, 1), RunID: runID, Agent: "runner", Kind: "run", Name: "first"},
		{Time: time.Unix(0, 2), RunID: runID, Agent: "runner", Kind: "model", Name: "second"},
	}
	for _, e := range events {
		b, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		key := "runs/" + runID + "/" + e.Time.Format("20060102150405.000000000") + "-" + e.Kind
		if err := scoped.Write(&store.Record{Key: key, Value: b}); err != nil {
			t.Fatal(err)
		}
	}

	got, err := LoadRunEvents(st, "runner", runID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d events, want 3", len(got))
	}
	for i, want := range []string{"first", "second", "third"} {
		if got[i].Name != want {
			t.Fatalf("event %d = %q, want %q (timeline: %#v)", i, got[i].Name, want, got)
		}
	}
}

func TestListRunSummaries(t *testing.T) {
	st := store.NewMemoryStore()
	scoped := store.Scope(st, "agent", "runner")
	events := []RunEvent{
		{Time: time.Unix(0, 1), RunID: "run-a", Agent: "runner", Kind: "run", Name: "first"},
		{Time: time.Unix(0, 2), RunID: "run-a", Agent: "runner", Kind: "tool", Name: "probe"},
		{Time: time.Unix(0, 3), RunID: "run-b", Agent: "runner", ParentID: "parent", Kind: "run", Name: "second"},
		{Time: time.Unix(0, 4), RunID: "run-b", Agent: "runner", ParentID: "parent", Kind: "error", Error: "boom"},
	}
	for _, e := range events {
		b, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		key := "runs/" + e.RunID + "/" + e.Time.Format("20060102150405.000000000") + "-" + e.Kind
		if err := scoped.Write(&store.Record{Key: key, Value: b}); err != nil {
			t.Fatal(err)
		}
	}

	got, err := ListRunSummaries(st, "runner")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d summaries, want 2: %#v", len(got), got)
	}
	if got[0].RunID != "run-a" || got[0].Events != 2 || got[0].LastKind != "tool" || !got[0].UpdatedAt.Equal(time.Unix(0, 2)) {
		t.Fatalf("unexpected run-a summary: %#v", got[0])
	}
	if got[1].RunID != "run-b" || got[1].ParentID != "parent" || got[1].Events != 2 || got[1].LastKind != "error" || got[1].LastError != "boom" {
		t.Fatalf("unexpected run-b summary: %#v", got[1])
	}
}
