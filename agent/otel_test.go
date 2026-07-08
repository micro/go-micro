package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/flow"
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
		if s.Name() == spanNameModelCall {
			if attrs[AttrAttempt] != "1" || attrs[AttrMaxAttempts] != "1" {
				t.Fatalf("model span missing attempt attributes: %#v", attrs)
			}
			if !spanEventHasRunInfo(s.Events(), "agent.model", runID, "runner") {
				t.Fatalf("model span missing model event: %#v", s.Events())
			}
		}
		if s.Name() == spanNameToolCall {
			if !spanEventHasRunInfo(s.Events(), "agent.tool", runID, "runner") {
				t.Fatalf("tool span missing tool event: %#v", s.Events())
			}
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

func TestAgentOpenTelemetryToolSpanIncludesWorkflowRunInfo(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exp))
	st := store.NewMemoryStore()
	a := New(Name("workflow-tool"), Provider("oteltest"), WithStore(st), TraceProvider(tp)).(*agentImpl)
	handler := a.traceTool(func(context.Context, ai.ToolCall) ai.ToolResult {
		return ai.ToolResult{Value: "ok"}
	})
	ctx := ai.WithRunInfo(context.Background(), ai.RunInfo{
		RunID:    "run-workflow-tool",
		ParentID: "parent-run",
		Agent:    "workflow-tool",
		Flow:     "deploy",
		Step:     "notify",
		Dispatch: "workflow",
		Trigger:  "manual",
	})

	res := handler(ctx, ai.ToolCall{ID: "call-1", Name: "notify", Input: map[string]any{"ok": true}})
	if resultError(res) != "" {
		t.Fatalf("tool returned error: %#v", res)
	}

	for _, span := range exp.GetSpans().Snapshots() {
		if span.Name() != spanNameToolCall {
			continue
		}
		attrs := spanAttributes(span.Attributes())
		if attrs[AttrRunID] != "run-workflow-tool" || attrs[AttrParentRunID] != "parent-run" || attrs[AttrAgentName] != "workflow-tool" {
			t.Fatalf("tool span missing run lineage: %#v", attrs)
		}
		if attrs[AttrFlowName] != "deploy" || attrs[AttrFlowStep] != "notify" || attrs[AttrDispatch] != "workflow" || attrs[AttrTrigger] != "manual" {
			t.Fatalf("tool span missing workflow run info: %#v", attrs)
		}
		return
	}
	t.Fatalf("tool span not emitted; got %d spans", len(exp.GetSpans().Snapshots()))
}

func TestAgentOpenTelemetryToolRetryAttempts(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exp))
	st := store.NewMemoryStore()
	calls := 0
	a := New(
		Name("tool-retry-otel"),
		Provider("oteltest"),
		WithStore(st),
		TraceProvider(tp),
		ToolRetry(3, time.Millisecond),
		WithTool("probe", "probe", nil, func(context.Context, map[string]any) (string, error) {
			calls++
			if calls == 1 {
				return "", errors.New("rate limit exceeded")
			}
			return "ok", nil
		}),
	)
	if _, err := a.Ask(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("tool calls = %d, want retry success after 2 attempts", calls)
	}

	var sawToolSpan bool
	for _, span := range exp.GetSpans().Snapshots() {
		if span.Name() != spanNameToolCall {
			continue
		}
		attrs := spanAttributes(span.Attributes())
		if attrs[AttrToolName] != "probe" {
			continue
		}
		if attrs[AttrToolAttempt] != "2" || attrs[AttrToolMaxAttempts] != "3" {
			t.Fatalf("tool retry span attempts = %#v", attrs)
		}
		if !spanEventHasAttr(span.Events(), "agent.tool", AttrToolAttempt, "2") || !spanEventHasAttr(span.Events(), "agent.tool", AttrToolMaxAttempts, "3") {
			t.Fatalf("tool retry event missing attempt attributes: %#v", span.Events())
		}
		sawToolSpan = true
	}
	if !sawToolSpan {
		t.Fatal("tool retry span not emitted")
	}

	summaries, err := ListRunSummaries(st, "tool-retry-otel")
	if err != nil {
		t.Fatal(err)
	}
	events, err := LoadRunEvents(st, "tool-retry-otel", summaries[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Kind == "tool" && event.Name == "probe" && event.Attempt == 2 && event.MaxAttempts == 3 {
			return
		}
	}
	t.Fatalf("persisted tool event missing retry attempts: %#v", events)
}

func spanEventHasAttr(events []trace.Event, name, key, value string) bool {
	for _, event := range events {
		if event.Name != name {
			continue
		}
		attrs := spanAttributes(event.Attributes)
		if attrs[key] == value {
			return true
		}
	}
	return false
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
		wantKind := strings.TrimPrefix(name, "agent.")
		if attrs[AttrRunID] == runID && attrs[AttrAgentName] == agentName && attrs[AttrRunEventKind] == wantKind {
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

func TestAgentCheckpointAndResumeTimelineEvents(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exp))
	st := store.NewMemoryStore()
	cp := flow.StoreCheckpoint(st, "resume-otel-agent")
	first := true
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		if first {
			first = false
			return nil, errors.New("temporary provider failure")
		}
		return &ai.Response{Reply: "resumed"}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("resume-otel-agent"), WithStore(st), WithCheckpoint(cp), TraceProvider(tp))
	_, err := a.Ask(context.Background(), "resume me")
	if err == nil {
		t.Fatal("Ask succeeded, want simulated failure")
	}

	runs, err := cp.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 {
		t.Fatalf("checkpointed runs = %d, want 1", len(runs))
	}
	resp, err := Resume(context.Background(), a, runs[0].ID)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if resp.Reply != "resumed" {
		t.Fatalf("reply = %q, want resumed", resp.Reply)
	}

	events, err := LoadRunEvents(st, "resume-otel-agent", runs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{"checkpoint": false, "resume": false}
	for _, e := range events {
		if _, ok := seen[e.Kind]; ok {
			seen[e.Kind] = true
		}
	}
	for kind, ok := range seen {
		if !ok {
			t.Fatalf("missing %s event in timeline: %#v", kind, events)
		}
	}

	var resumeSpanEvent bool
	for _, s := range exp.GetSpans().Snapshots() {
		if s.Name() != spanNameRun {
			continue
		}
		for _, e := range s.Events() {
			if e.Name == "agent.resume" {
				resumeSpanEvent = true
			}
		}
	}
	if !resumeSpanEvent {
		t.Fatal("run span missing agent.resume event")
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
		{Time: time.Unix(0, 4), RunID: "run-b", Agent: "runner", ParentID: "parent", Kind: "checkpoint", Name: "ask", Status: "failed"},
		{Time: time.Unix(0, 5), RunID: "run-b", Agent: "runner", ParentID: "parent", Kind: "error", Error: "context deadline exceeded", ErrorKind: string(ai.ErrorKindTimeout)},
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
	if got[1].RunID != "run-b" || got[1].ParentID != "parent" || got[1].Events != 3 || got[1].Status != "timeout" || got[1].DurationMS != 0 || got[1].LastKind != "error" || got[1].Checkpoint != "failed" || got[1].Stage != "ask" || got[1].LastError != "context deadline exceeded" || got[1].LastErrorKind != string(ai.ErrorKindTimeout) {
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

type otelStreamModel struct{ opts ai.Options }

func (m *otelStreamModel) Init(opts ...ai.Option) error {
	for _, o := range opts {
		o(&m.opts)
	}
	return nil
}
func (m *otelStreamModel) Options() ai.Options { return m.opts }
func (m *otelStreamModel) String() string      { return "otelstream" }
func (m *otelStreamModel) Generate(context.Context, *ai.Request, ...ai.GenerateOption) (*ai.Response, error) {
	return &ai.Response{Reply: "unused"}, nil
}
func (m *otelStreamModel) Stream(context.Context, *ai.Request, ...ai.GenerateOption) (ai.Stream, error) {
	return &otelTestStream{chunks: []*ai.Response{{Reply: "one", Usage: ai.Usage{InputTokens: 1, OutputTokens: 2, TotalTokens: 3}}, {Reply: "two", Usage: ai.Usage{InputTokens: 1, OutputTokens: 4, TotalTokens: 5}}}}, nil
}

type otelTestStream struct {
	chunks []*ai.Response
	idx    int
}

func (s *otelTestStream) Recv() (*ai.Response, error) {
	if s.idx >= len(s.chunks) {
		return nil, io.EOF
	}
	resp := s.chunks[s.idx]
	s.idx++
	return resp, nil
}

func (s *otelTestStream) Close() error { return nil }

func TestAgentOpenTelemetrySpansModelStream(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exp))
	st := store.NewMemoryStore()
	a := New(Name("stream-runner"), Provider("oteltest"), Model("stream-model"), WithStore(st), TraceProvider(tp))
	m := a.(*agentImpl).tracedModel(&otelStreamModel{opts: ai.Options{Model: "stream-model"}})
	ctx := ai.WithRunInfo(context.Background(), ai.RunInfo{RunID: "stream-run-1", ParentID: "parent-run", Agent: "stream-runner", Attempt: 2, MaxAttempts: 3, Flow: "deploy", Step: "plan"})

	stream, err := m.Stream(ctx, &ai.Request{Prompt: "stream"})
	if err != nil {
		t.Fatal(err)
	}
	for {
		_, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}

	spans := exp.GetSpans().Snapshots()
	var sawStream bool
	for _, s := range spans {
		if s.Name() != spanNameModelStream {
			continue
		}
		attrs := spanAttributes(s.Attributes())
		if attrs[AttrRunID] != "stream-run-1" || attrs[AttrParentRunID] != "parent-run" || attrs[AttrAgentName] != "stream-runner" {
			t.Fatalf("stream span missing run lineage: %#v", attrs)
		}
		if attrs[AttrFlowName] != "deploy" || attrs[AttrFlowStep] != "plan" {
			t.Fatalf("stream span missing workflow attributes: %#v", attrs)
		}
		if attrs[AttrAttempt] != "2" || attrs[AttrMaxAttempts] != "3" || attrs[AttrTotalTokens] != "5" {
			t.Fatalf("stream span missing attempt/usage attributes: %#v", attrs)
		}
		if !spanEventHasRunInfo(s.Events(), "agent.stream", "stream-run-1", "stream-runner") {
			t.Fatalf("stream span missing stream event: %#v", s.Events())
		}
		sawStream = true
	}
	if !sawStream {
		t.Fatalf("stream span not emitted; got %d spans", len(spans))
	}

	events, err := LoadRunEvents(st, "stream-runner", "stream-run-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Kind != "stream" || events[0].TraceID == "" || events[0].SpanID == "" || events[0].Tokens.TotalTokens != 5 {
		t.Fatalf("unexpected stream run event: %#v", events)
	}
}
