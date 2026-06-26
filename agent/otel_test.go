package agent

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/store"
	"go.opentelemetry.io/otel/attribute"
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
	var runID string
	for _, s := range spans {
		if _, ok := want[s.Name()]; ok {
			want[s.Name()] = true
		}
		attrs := spanAttributes(s.Attributes())
		if s.Name() == spanNameRun {
			runID = attrs[AttrRunID]
		}
	}
	for name, seen := range want {
		if !seen {
			t.Fatalf("span %s not emitted; got %d spans", name, len(spans))
		}
	}
	if runID == "" {
		t.Fatal("run span missing run id attribute")
	}
	for _, s := range spans {
		if s.Name() != spanNameModelCall && s.Name() != spanNameToolCall {
			continue
		}
		attrs := spanAttributes(s.Attributes())
		if attrs[AttrRunID] != runID || attrs[AttrAgentName] != "runner" {
			t.Fatalf("%s missing run correlation attributes: %#v", s.Name(), attrs)
		}
	}
	keys, err := store.Scope(st, "agent", "runner").List(store.ListPrefix("runs/"))
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) == 0 {
		t.Fatal("expected run events to be recorded")
	}
	summaries, err := ListRunSummaries(st, "runner")
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 {
		t.Fatalf("got %d summaries, want 1", len(summaries))
	}
	if summaries[0].LastKind != "done" {
		t.Fatalf("LastKind = %q, want done", summaries[0].LastKind)
	}
	if summaries[0].Status != "done" {
		t.Fatalf("Status = %q, want done", summaries[0].Status)
	}
	if summaries[0].DurationMS < 0 {
		t.Fatalf("DurationMS = %d, want non-negative", summaries[0].DurationMS)
	}
	if summaries[0].TraceID == "" || summaries[0].SpanID == "" {
		t.Fatalf("summary missing trace correlation: %#v", summaries[0])
	}
	events, err := LoadRunEvents(st, "runner", summaries[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 || events[0].TraceID == "" || events[0].SpanID == "" {
		t.Fatalf("events missing trace correlation: %#v", events)
	}
}

func spanAttributes(attrs []attribute.KeyValue) map[string]string {
	out := make(map[string]string, len(attrs))
	for _, attr := range attrs {
		out[string(attr.Key)] = attr.Value.AsString()
	}
	return out
}

func TestAgentRunTimelineRecordsModelAndToolWithoutTraceProvider(t *testing.T) {
	st := store.NewMemoryStore()
	a := New(Name("runner-noop"), Provider("oteltest"), WithStore(st), WithTool("probe", "probe", nil, func(context.Context, map[string]any) (string, error) { return "ok", nil }))
	if _, err := a.Ask(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	keys, err := store.Scope(st, "agent", "runner-noop").List(store.ListPrefix("runs/"))
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) == 0 {
		t.Fatal("expected run timeline without TraceProvider")
	}
	summaries, err := ListRunSummaries(st, "runner-noop")
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 {
		t.Fatalf("got %d summaries, want 1", len(summaries))
	}
	if summaries[0].Status != "done" || summaries[0].LastKind != "done" {
		t.Fatalf("unexpected summary without TraceProvider: %#v", summaries[0])
	}
	if summaries[0].TraceID != "" || summaries[0].SpanID != "" {
		t.Fatalf("unexpected trace correlation without TraceProvider: %#v", summaries[0])
	}
	events, err := LoadRunEvents(st, "runner-noop", summaries[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{"run": false, "model": false, "tool": false, "done": false}
	for _, e := range events {
		seen[e.Kind] = true
		if e.TraceID != "" || e.SpanID != "" {
			t.Fatalf("event has trace correlation without TraceProvider: %#v", e)
		}
	}
	for kind, ok := range seen {
		if !ok {
			t.Fatalf("missing %s event in timeline: %#v", kind, events)
		}
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
		{Time: time.Unix(0, 1), RunID: "run-a", Agent: "runner", TraceID: "trace-a", SpanID: "span-a", Kind: "run", Name: "first"},
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
	if got[0].RunID != "run-a" || got[0].TraceID != "trace-a" || got[0].SpanID != "span-a" || got[0].Events != 2 || got[0].Status != "running" || got[0].DurationMS != 0 || got[0].LastKind != "tool" || !got[0].UpdatedAt.Equal(time.Unix(0, 2)) {
		t.Fatalf("unexpected run-a summary: %#v", got[0])
	}
	if got[1].RunID != "run-b" || got[1].ParentID != "parent" || got[1].Events != 2 || got[1].Status != "error" || got[1].DurationMS != 0 || got[1].LastKind != "error" || got[1].LastError != "boom" {
		t.Fatalf("unexpected run-b summary: %#v", got[1])
	}
}

func TestListRunSummariesWithOptionsFiltersAndLimits(t *testing.T) {
	st := store.NewMemoryStore()
	scoped := store.Scope(st, "agent", "runner")
	events := []RunEvent{
		{Time: time.Unix(0, 1), RunID: "run-old", Agent: "runner", Kind: "run"},
		{Time: time.Unix(0, 2), RunID: "run-old", Agent: "runner", Kind: "done"},
		{Time: time.Unix(0, 3), RunID: "run-new", Agent: "runner", TraceID: "abcdef1234567890", Kind: "run"},
		{Time: time.Unix(0, 4), RunID: "run-new", Agent: "runner", Kind: "error", Error: "boom"},
	}
	for _, e := range events {
		b, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		if err := scoped.Write(&store.Record{Key: "runs/" + e.RunID + "/" + e.Time.Format("20060102150405.000000000") + "-" + e.Kind, Value: b}); err != nil {
			t.Fatal(err)
		}
	}

	got, err := ListRunSummariesWithOptions(st, "runner", RunListOptions{Status: "error", TraceID: "abcdef", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].RunID != "run-new" || got[0].Status != "error" {
		t.Fatalf("filtered summaries = %#v", got)
	}
}
