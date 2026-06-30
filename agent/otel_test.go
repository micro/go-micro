package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/store"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

const codesError = codes.Error

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
		if strings.Contains(req.Prompt, "delegate") {
			_ = m.opts.ToolHandler(ctx, ai.ToolCall{ID: "call-delegate", Name: toolDelegate, Input: map[string]any{"task": "subtask"}})
		} else if !strings.Contains(req.Prompt, "subtask") {
			_ = m.opts.ToolHandler(ctx, ai.ToolCall{ID: "call-1", Name: "probe", Input: map[string]any{"ok": true}})
		}
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
	var runEvents []trace.Event
	for _, s := range spans {
		if s.Name() == spanNameRun {
			runEvents = s.Events()
			break
		}
	}
	if !spanEventHasRunInfo(runEvents, "agent.run", runID, "runner") || !spanEventHasRunInfo(runEvents, "agent.done", runID, "runner") {
		t.Fatalf("run span missing run-info events: %#v", runEvents)
	}
	for _, s := range spans {
		if s.Name() != spanNameModelCall && s.Name() != spanNameToolCall {
			continue
		}
		attrs := spanAttributes(s.Attributes())
		if attrs[AttrRunID] != runID || attrs[AttrAgentName] != "runner" {
			t.Fatalf("%s missing run correlation attributes: %#v", s.Name(), attrs)
		}
		if s.Name() == spanNameModelCall && (attrs[AttrAttempt] != "1" || attrs[AttrMaxAttempts] != "1") {
			t.Fatalf("model span missing attempt attributes: %#v", attrs)
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

func TestAgentRunObservabilityRedactsInputByDefault(t *testing.T) {
	secret := "deploy production with token sk-secret"
	exp := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exp))
	st := store.NewMemoryStore()
	a := New(Name("redactor"), Provider("oteltest"), WithStore(st), TraceProvider(tp))
	if _, err := a.Ask(context.Background(), secret); err != nil {
		t.Fatal(err)
	}

	spans := exp.GetSpans().Snapshots()
	var sawInputChars bool
	for _, s := range spans {
		for _, event := range s.Events() {
			attrs := spanAttributes(event.Attributes)
			if attrs["agent.event.name"] == secret {
				t.Fatalf("span event leaked raw input: %#v", event)
			}
			if attrs[AttrInputChars] == fmt.Sprint(len(secret)) {
				sawInputChars = true
			}
		}
	}
	if !sawInputChars {
		t.Fatal("run event missing redacted input length attribute")
	}

	summaries, err := ListRunSummaries(st, "redactor")
	if err != nil {
		t.Fatal(err)
	}
	events, err := LoadRunEvents(st, "redactor", summaries[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Name == secret {
			t.Fatalf("persisted run event leaked raw input: %#v", event)
		}
		if event.Kind == "run" && event.InputChars != len(secret) {
			t.Fatalf("run event InputChars = %d, want %d", event.InputChars, len(secret))
		}
	}
}

func TestAgentTraceInputsOptInRecordsInput(t *testing.T) {
	message := "operator-approved diagnostic prompt"
	st := store.NewMemoryStore()
	a := New(Name("input-opt-in"), Provider("oteltest"), WithStore(st), TraceInputs(true))
	if _, err := a.Ask(context.Background(), message); err != nil {
		t.Fatal(err)
	}

	summaries, err := ListRunSummaries(st, "input-opt-in")
	if err != nil {
		t.Fatal(err)
	}
	events, err := LoadRunEvents(st, "input-opt-in", summaries[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Kind == "run" && event.Name == message {
			return
		}
	}
	t.Fatalf("opt-in run event did not record message: %#v", events)
}

type failingOtelModel struct{ opts ai.Options }

func (m *failingOtelModel) Init(opts ...ai.Option) error {
	for _, o := range opts {
		o(&m.opts)
	}
	return nil
}
func (m *failingOtelModel) Options() ai.Options { return m.opts }
func (m *failingOtelModel) String() string      { return "otelfail" }
func (m *failingOtelModel) Stream(context.Context, *ai.Request, ...ai.GenerateOption) (ai.Stream, error) {
	return nil, nil
}
func (m *failingOtelModel) Generate(context.Context, *ai.Request, ...ai.GenerateOption) (*ai.Response, error) {
	return nil, errors.New("provider exploded")
}

func init() {
	ai.Register("otelfail", func(opts ...ai.Option) ai.Model { return &failingOtelModel{opts: ai.NewOptions(opts...)} })
}

func TestAgentOpenTelemetrySpansModelFailure(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exp))
	st := store.NewMemoryStore()
	a := New(Name("failing-runner"), Provider("otelfail"), WithStore(st), TraceProvider(tp))
	if _, err := a.Ask(context.Background(), "hello"); err == nil {
		t.Fatal("Ask succeeded, want provider error")
	}

	spans := exp.GetSpans().Snapshots()
	var sawRunError, sawModelError bool
	for _, s := range spans {
		attrs := spanAttributes(s.Attributes())
		switch s.Name() {
		case spanNameRun:
			if attrs[AttrAgentName] == "failing-runner" && s.Status().Code == codesError {
				sawRunError = true
			}
		case spanNameModelCall:
			if attrs[AttrAgentName] == "failing-runner" && attrs[AttrAttempt] == "1" && attrs[AttrErrorKind] == string(ai.ErrorKindUnknown) && s.Status().Code == codesError {
				sawModelError = true
			}
		}
	}
	if !sawRunError || !sawModelError {
		t.Fatalf("missing error spans: run=%v model=%v spans=%d", sawRunError, sawModelError, len(spans))
	}

	summaries, err := ListRunSummaries(st, "failing-runner")
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 || summaries[0].Status != "error" || summaries[0].LastError == "" {
		t.Fatalf("unexpected failure summary: %#v", summaries)
	}
	events, err := LoadRunEvents(st, "failing-runner", summaries[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	var sawModelEvent bool
	for _, event := range events {
		if event.Kind == "model" && event.Attempt == 1 && event.MaxAttempts == 1 && event.Error != "" && event.ErrorKind == string(ai.ErrorKindUnknown) {
			sawModelEvent = true
		}
	}
	if !sawModelEvent {
		t.Fatalf("missing failed model event with attempt metadata: %#v", events)
	}
}

func spanEventHasRunInfo(events []trace.Event, name, runID, agentName string) bool {
	for _, event := range events {
		if event.Name != name {
			continue
		}
		attrs := spanAttributes(event.Attributes)
		if attrs[AttrRunID] == runID && attrs[AttrAgentName] == agentName {
			return true
		}
	}
	return false
}

func spanAttributes(attrs []attribute.KeyValue) map[string]string {
	out := make(map[string]string, len(attrs))
	for _, attr := range attrs {
		out[string(attr.Key)] = fmt.Sprint(attr.Value.AsInterface())
	}
	return out
}

func TestAgentOpenTelemetrySpansDelegateLineage(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exp))
	st := store.NewMemoryStore()
	a := New(Name("conductor"), Provider("oteltest"), WithStore(st), TraceProvider(tp))
	if _, err := a.Ask(context.Background(), "delegate please"); err != nil {
		t.Fatal(err)
	}

	spans := exp.GetSpans().Snapshots()
	var parentRunID string
	var delegateSpanID string
	var subRunSeen bool
	for _, s := range spans {
		attrs := spanAttributes(s.Attributes())
		if s.Name() == spanNameRun && attrs[AttrAgentName] == "conductor" {
			parentRunID = attrs[AttrRunID]
		}
	}
	if parentRunID == "" {
		t.Fatal("parent run span missing run id")
	}
	for _, s := range spans {
		attrs := spanAttributes(s.Attributes())
		if s.Name() == spanNameToolCall && attrs[AttrToolName] == toolDelegate {
			if attrs[AttrDelegate] != "true" || attrs[AttrRunID] != parentRunID {
				t.Fatalf("delegate span missing correlation attributes: %#v", attrs)
			}
			delegateSpanID = s.SpanContext().SpanID().String()
		}
	}
	if delegateSpanID == "" {
		t.Fatal("delegate tool span not emitted")
	}
	for _, s := range spans {
		attrs := spanAttributes(s.Attributes())
		if s.Name() == spanNameRun && attrs[AttrAgentName] == "conductor.sub" {
			if attrs[AttrParentRunID] != parentRunID {
				t.Fatalf("sub-agent run parent attr = %q, want %q", attrs[AttrParentRunID], parentRunID)
			}
			if s.Parent().SpanID().String() != delegateSpanID {
				t.Fatalf("sub-agent run parent span = %s, want delegate span %s", s.Parent().SpanID(), delegateSpanID)
			}
			subRunSeen = true
		}
	}
	if !subRunSeen {
		t.Fatalf("sub-agent run span not emitted; got %d spans", len(spans))
	}
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
		{Time: time.Unix(0, 4), RunID: "run-b", Agent: "runner", ParentID: "parent", Kind: "error", Error: "context deadline exceeded", ErrorKind: string(ai.ErrorKindTimeout)},
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
	if got[1].RunID != "run-b" || got[1].ParentID != "parent" || got[1].Events != 2 || got[1].Status != "timeout" || got[1].DurationMS != 0 || got[1].LastKind != "error" || got[1].LastError != "context deadline exceeded" || got[1].LastErrorKind != string(ai.ErrorKindTimeout) {
		t.Fatalf("unexpected run-b summary: %#v", got[1])
	}
}

func TestRunStatusClassifiesOperationalErrorKinds(t *testing.T) {
	tests := []struct {
		name string
		kind ai.ErrorKind
		want string
	}{
		{name: "canceled", kind: ai.ErrorKindCanceled, want: "canceled"},
		{name: "timeout", kind: ai.ErrorKindTimeout, want: "timeout"},
		{name: "rate limited", kind: ai.ErrorKindRateLimited, want: "rate_limited"},
		{name: "provider", kind: ai.ErrorKindProvider, want: "error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runStatus([]RunEvent{
				{Kind: "run"},
				{Kind: "error", Error: "failed", ErrorKind: string(tt.kind)},
			})
			if got != tt.want {
				t.Fatalf("runStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestListRunSummariesWithOptionsFiltersAndLimits(t *testing.T) {
	st := store.NewMemoryStore()
	scoped := store.Scope(st, "agent", "runner")
	events := []RunEvent{
		{Time: time.Unix(0, 1), RunID: "run-old", Agent: "runner", Kind: "run"},
		{Time: time.Unix(0, 2), RunID: "run-old", Agent: "runner", Kind: "done"},
		{Time: time.Unix(0, 3), RunID: "run-new", Agent: "runner", TraceID: "abcdef1234567890", Kind: "run"},
		{Time: time.Unix(0, 4), RunID: "run-new", Agent: "runner", Kind: "error", Error: "rate limit exceeded", ErrorKind: string(ai.ErrorKindRateLimited)},
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

	got, err := ListRunSummariesWithOptions(st, "runner", RunListOptions{Status: "rate_limited", TraceID: "abcdef", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].RunID != "run-new" || got[0].Status != "rate_limited" {
		t.Fatalf("filtered summaries = %#v", got)
	}
}
