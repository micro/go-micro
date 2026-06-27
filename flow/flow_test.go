package flow

import (
	"context"
	"testing"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/registry"
)

func TestNew(t *testing.T) {
	f := New("test-flow",
		Trigger("events.test"),
		Prompt("Handle this: {{.Data}}"),
		Provider("anthropic"),
		APIKey("test-key"),
		HistoryLimit(10),
	)

	if f.Name() != "test-flow" {
		t.Errorf("name = %q, want test-flow", f.Name())
	}
	if f.opts.TriggerTopic != "events.test" {
		t.Errorf("trigger = %q", f.opts.TriggerTopic)
	}
	if f.opts.Provider != "anthropic" {
		t.Errorf("provider = %q", f.opts.Provider)
	}
	if f.opts.HistoryLimit != 10 {
		t.Errorf("history limit = %d", f.opts.HistoryLimit)
	}
	if f.tmpl == nil {
		t.Fatal("template not parsed")
	}
}

func TestPromptTemplate(t *testing.T) {
	f := New("tmpl-test",
		Prompt("User created: {{.Data}}. Send welcome email."),
	)

	// Test that the template renders
	if f.tmpl == nil {
		t.Fatal("template not parsed")
	}
}

func TestResultsEmpty(t *testing.T) {
	f := New("empty")
	results := f.Results()
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestOnResultCallback(t *testing.T) {
	var called bool
	f := New("callback",
		OnResult(func(r Result) {
			called = true
			if r.FlowName != "callback" {
				t.Errorf("flow name = %q", r.FlowName)
			}
		}),
	)

	f.record(Result{FlowName: "callback"})

	if !called {
		t.Error("OnResult not called")
	}
	if len(f.Results()) != 1 {
		t.Errorf("results = %d, want 1", len(f.Results()))
	}
}

func TestDefaultOptions(t *testing.T) {
	f := New("defaults")

	if f.opts.Provider != "openai" {
		t.Errorf("default provider = %q, want openai", f.opts.Provider)
	}
	if f.opts.HistoryLimit != 20 {
		t.Errorf("default history limit = %d, want 20", f.opts.HistoryLimit)
	}
	if f.opts.SystemPrompt == "" {
		t.Error("default system prompt is empty")
	}
}

func TestSingleStepFlowRunInfoIdentifiesFlow(t *testing.T) {
	model := &runInfoModel{}
	f := New("single-observed")
	f.model = model
	f.toolSet = ai.NewTools(registry.NewMemoryRegistry())

	if err := f.Execute(context.Background(), "observe me"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if model.got.RunID == "" {
		t.Fatal("RunInfo.RunID is empty")
	}
	if model.got.Flow != "single-observed" {
		t.Fatalf("RunInfo.Flow = %q, want single-observed", model.got.Flow)
	}
	if model.got.Agent != "" {
		t.Fatalf("RunInfo.Agent = %q, want empty for flow-owned LLM run", model.got.Agent)
	}
	if model.got.Step != "" {
		t.Fatalf("RunInfo.Step = %q, want empty for single-step flow", model.got.Step)
	}
}

type runInfoModel struct {
	got ai.RunInfo
}

func (m *runInfoModel) Init(...ai.Option) error { return nil }

func (m *runInfoModel) Options() ai.Options { return ai.Options{} }

func (m *runInfoModel) Generate(ctx context.Context, _ *ai.Request, _ ...ai.GenerateOption) (*ai.Response, error) {
	m.got, _ = ai.RunInfoFrom(ctx)
	return &ai.Response{Reply: "ok"}, nil
}

func (m *runInfoModel) Stream(context.Context, *ai.Request, ...ai.GenerateOption) (ai.Stream, error) {
	return nil, ai.ErrStreamingUnsupported
}

func (m *runInfoModel) String() string { return "run-info-model" }
