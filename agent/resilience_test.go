package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/flow"
	"go-micro.dev/v6/store"
)

func TestAskCancellationAbortsPromptly(t *testing.T) {
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("cancel"), ModelCallTimeout(time.Second), ModelRetry(3, time.Millisecond))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_, err := a.Ask(ctx, "stop")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Ask error = %v, want context canceled", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("Ask took %s after cancellation, want prompt abort", elapsed)
	}
}

func TestAskRetriesTransientErrorsThenSucceeds(t *testing.T) {
	attempts := 0
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		attempts++
		if attempts < 3 {
			return nil, context.DeadlineExceeded
		}
		return &ai.Response{Reply: "ok"}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("retry-success"), ModelRetry(3, time.Millisecond))
	resp, err := a.Ask(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Ask returned error: %v", err)
	}
	if resp.Reply != "ok" {
		t.Fatalf("reply = %q, want ok", resp.Reply)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestAskRetriesTransientErrorsThenSurfacesStructuredError(t *testing.T) {
	attempts := 0
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		attempts++
		return nil, context.DeadlineExceeded
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("retry-fail"), ModelRetry(2, time.Millisecond))
	_, err := a.Ask(context.Background(), "hello")
	var retryErr *ai.RetryError
	if !errors.As(err, &retryErr) {
		t.Fatalf("Ask error = %T %v, want *ai.RetryError", err, err)
	}
	if retryErr.Attempts != 2 {
		t.Fatalf("retry attempts = %d, want 2", retryErr.Attempts)
	}
	if attempts != 2 {
		t.Fatalf("model attempts = %d, want 2", attempts)
	}
	if !strings.Contains(err.Error(), "micro inspect agent <name> --status timeout") ||
		!strings.Contains(err.Error(), "docs/guides/debugging-agents.md") {
		t.Fatalf("Ask error = %q, want actionable timeout/debugging guidance", err.Error())
	}
}

func TestModelRetryDoesNotDuplicateCheckpointedToolSideEffects(t *testing.T) {
	ctx := context.Background()
	cp := flow.StoreCheckpoint(store.NewMemoryStore(), "retry-tool-dedupe-agent")
	attempts := 0
	toolRuns := 0
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		attempts++
		if opts.ToolHandler == nil {
			t.Fatal("missing tool handler")
		}
		res := opts.ToolHandler(ctx, ai.ToolCall{ID: "create-1", Name: "external.create", Input: map[string]any{"title": "Retry safe"}})
		if res.Content != "created Retry safe" {
			t.Fatalf("tool result = %q, want cached create result", res.Content)
		}
		if attempts == 1 {
			return nil, testStatusError{code: 503}
		}
		return &ai.Response{Reply: "done", ToolCalls: []ai.ToolCall{{ID: "create-1", Name: "external.create", Input: map[string]any{"title": "Retry safe"}, Result: res.Content}}}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(
		Name("retry-tool-dedupe-agent"),
		WithCheckpoint(cp),
		ModelRetry(2, time.Millisecond),
		WithTool("external.create", "create once", nil, func(context.Context, map[string]any) (string, error) {
			toolRuns++
			return "created Retry safe", nil
		}),
	)

	resp, err := a.Ask(ctx, "create once despite a transient provider retry")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if resp.Reply != "done" {
		t.Fatalf("reply = %q, want done", resp.Reply)
	}
	if attempts != 2 {
		t.Fatalf("model attempts = %d, want retry after transient provider failure", attempts)
	}
	if toolRuns != 1 {
		t.Fatalf("tool executions = %d, want checkpointed side effect reused across retry", toolRuns)
	}
	runs, err := cp.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("checkpointed runs = %d, want 1", len(runs))
	}
	if _, ok := findStep(runs[0].Steps, `tool:external.create:{"title":"Retry safe"}`); !ok {
		t.Fatalf("checkpoint steps = %#v, want completed external.create step", runs[0].Steps)
	}
}

func TestAskRateLimitFailureSuggestsPreflightAndInspect(t *testing.T) {
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		return nil, testStatusError{code: 429}
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("rate-limit-guidance"), ModelRetry(1, time.Millisecond))
	_, err := a.Ask(context.Background(), "hello")
	if err == nil {
		t.Fatal("Ask succeeded, want rate-limit failure")
	}
	if !strings.Contains(err.Error(), "micro inspect agent <name> --status rate_limited") ||
		!strings.Contains(err.Error(), "micro agent preflight") {
		t.Fatalf("Ask error = %q, want inspect and preflight guidance", err.Error())
	}
	if ai.ClassifyError(err) != ai.ErrorKindRateLimited {
		t.Fatalf("ClassifyError(wrapped error) = %q, want rate_limited", ai.ClassifyError(err))
	}
}

func TestCanceledAskContextSkipsToolExecution(t *testing.T) {
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		if opts.ToolHandler == nil {
			t.Fatal("missing tool handler")
		}
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		res := opts.ToolHandler(canceled, ai.ToolCall{ID: "call-1", Name: toolPlan, Input: map[string]any{
			"steps": []any{map[string]any{"task": "should not persist", "status": "pending"}},
		}})
		if !strings.Contains(res.Content, context.Canceled.Error()) {
			t.Fatalf("tool result = %q, want cancellation error", res.Content)
		}
		return &ai.Response{Reply: "ok"}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("cancel-tools"))
	if _, err := a.Ask(context.Background(), "try a canceled tool"); err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if plan := a.loadPlan(); plan != "" {
		t.Fatalf("plan persisted after canceled tool context: %q", plan)
	}
}

func TestToolCallTimeoutPropagatesDeadlineToCustomTool(t *testing.T) {
	var sawDeadline bool
	a := newTestAgent(
		Name("tool-timeout"),
		ToolCallTimeout(10*time.Millisecond),
		WithTool("slow", "slow tool", nil, func(ctx context.Context, input map[string]any) (string, error) {
			if _, ok := ctx.Deadline(); ok {
				sawDeadline = true
			}
			<-ctx.Done()
			return "", ctx.Err()
		}),
	)

	start := time.Now()
	content := toolContent(a.toolHandler(), "slow", nil)
	if !sawDeadline {
		t.Fatal("custom tool did not receive a deadline")
	}
	if !strings.Contains(content, context.DeadlineExceeded.Error()) {
		t.Fatalf("tool result = %q, want deadline exceeded", content)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("tool call took %s, want bounded timeout", elapsed)
	}
}

func TestAskCancellationDuringToolCallFailsRun(t *testing.T) {
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		if opts.ToolHandler == nil {
			t.Fatal("missing tool handler")
		}
		res := opts.ToolHandler(ctx, ai.ToolCall{ID: "call-1", Name: "cancel-self"})
		if !strings.Contains(res.Content, context.Canceled.Error()) {
			t.Fatalf("tool result = %q, want cancellation error", res.Content)
		}
		return &ai.Response{Reply: "should not succeed"}, nil
	}
	defer func() { fakeGen = nil }()

	ctx, cancel := context.WithCancel(context.Background())
	a := newTestAgent(
		Name("cancel-during-tool"),
		WithTool("cancel-self", "cancel the run context", nil, func(context.Context, map[string]any) (string, error) {
			cancel()
			return "", context.Canceled
		}),
	)

	_, err := a.Ask(ctx, "cancel during tool")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Ask error = %v, want context canceled", err)
	}
}

func TestSlowProviderTimeoutPreventsLateToolSideEffects(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		close(started)
		<-release
		defer close(done)
		if opts.ToolHandler == nil {
			t.Fatal("missing tool handler")
		}
		res := opts.ToolHandler(ctx, ai.ToolCall{ID: "late-1", Name: "external.create", Input: map[string]any{"title": "too late"}})
		if !strings.Contains(res.Content, context.DeadlineExceeded.Error()) {
			t.Errorf("late tool result = %q, want deadline exceeded", res.Content)
		}
		return &ai.Response{Reply: "late", ToolCalls: []ai.ToolCall{{ID: "late-1", Name: "external.create", Input: map[string]any{"title": "too late"}, Result: res.Content}}}, nil
	}
	defer func() { fakeGen = nil }()

	toolRuns := 0
	a := newTestAgent(
		Name("slow-provider-late-tool"),
		ModelCallTimeout(10*time.Millisecond),
		WithTool("external.create", "create once", nil, func(context.Context, map[string]any) (string, error) {
			toolRuns++
			return "created", nil
		}),
	)

	_, err := a.Ask(context.Background(), "provider times out before tool")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Ask error = %v, want deadline exceeded", err)
	}
	select {
	case <-started:
	default:
		t.Fatal("provider was not called")
	}
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("late provider call did not finish")
	}
	if toolRuns != 0 {
		t.Fatalf("late tool executions = %d, want 0", toolRuns)
	}
}

func TestAskCheckpointRecordsTerminalOperationalFailureStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "canceled", err: context.Canceled, want: "canceled"},
		{name: "timeout", err: context.DeadlineExceeded, want: "timeout"},
		{name: "rate limited", err: testStatusError{code: 429}, want: "rate_limited"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := flow.StoreCheckpoint(store.NewMemoryStore(), "terminal-"+strings.ReplaceAll(tt.name, " ", "-"))
			fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
				return nil, tt.err
			}
			defer func() { fakeGen = nil }()

			a := newTestAgent(Name("terminal-"+strings.ReplaceAll(tt.name, " ", "-")), WithCheckpoint(cp))
			_, err := a.Ask(context.Background(), "fail safely")
			if err == nil {
				t.Fatal("Ask succeeded, want failure")
			}

			runs, err := cp.List(context.Background())
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(runs) != 1 {
				t.Fatalf("checkpointed runs = %d, want 1", len(runs))
			}
			if runs[0].Status != tt.want {
				t.Fatalf("run status = %q, want %q", runs[0].Status, tt.want)
			}
			if len(runs[0].Steps) == 0 || runs[0].Steps[0].Status != tt.want {
				t.Fatalf("step status = %#v, want %q", runs[0].Steps, tt.want)
			}
			if pending, err := Pending(context.Background(), a); err != nil || len(pending) != 0 {
				t.Fatalf("Pending = %#v, %v; want no terminal run", pending, err)
			}
		})
	}
}

type testStatusError struct {
	code int
}

func (e testStatusError) Error() string { return "provider status error" }

func (e testStatusError) StatusCode() int { return e.code }

func TestToolRetryRetriesTransientToolErrorsThenSucceeds(t *testing.T) {
	attempts := 0
	a := newTestAgent(
		Name("tool-retry-success"),
		ToolRetry(3, time.Millisecond),
		WithTool("flaky", "flaky tool", nil, func(context.Context, map[string]any) (string, error) {
			attempts++
			if attempts < 3 {
				return "", context.DeadlineExceeded
			}
			return "ok", nil
		}),
	)

	content := toolContent(a.toolHandler(), "flaky", nil)
	if content != "ok" {
		t.Fatalf("tool result = %q, want ok", content)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestToolRetryDoesNotRetryGuardrailRefusals(t *testing.T) {
	attempts := 0
	a := newTestAgent(
		Name("tool-retry-refusal"),
		MaxSteps(1),
		ToolRetry(3, time.Millisecond),
		WithTool("counted", "counted tool", nil, func(context.Context, map[string]any) (string, error) {
			attempts++
			return "ok", nil
		}),
	)
	h := a.toolHandler()
	_ = toolContent(h, "counted", nil)
	content := toolContent(h, "counted", nil)
	if !strings.Contains(content, "step limit reached") {
		t.Fatalf("tool result = %q, want step-limit refusal", content)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want only the allowed tool call to execute", attempts)
	}
}
