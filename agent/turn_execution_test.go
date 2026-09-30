package agent

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	pb "go-micro.dev/v6/agent/proto"
	flow "go-micro.dev/v6/internal/runstate"
	"go-micro.dev/v6/metadata"
	"go-micro.dev/v6/model"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
)

type turnTestModel struct {
	opts model.Options
	turn func(context.Context, *model.Request) (*model.Response, error)
}

func (m *turnTestModel) Init(opts ...model.Option) error {
	for _, o := range opts {
		o(&m.opts)
	}
	return nil
}
func (m *turnTestModel) Options() model.Options { return m.opts }
func (m *turnTestModel) String() string         { return "turn-test" }
func (m *turnTestModel) Generate(context.Context, *model.Request, ...model.GenerateOption) (*model.Response, error) {
	return nil, errors.New("legacy Generate must not be called")
}
func (m *turnTestModel) Stream(context.Context, *model.Request, ...model.GenerateOption) (model.Stream, error) {
	return nil, model.ErrStreamingUnsupported
}
func (m *turnTestModel) Turn(ctx context.Context, r *model.Request, _ ...model.GenerateOption) (*model.Response, error) {
	return m.turn(ctx, r)
}

func TestTurnRetryDoesNotReplayTools(t *testing.T) {
	effects, followups := 0, 0
	name := t.Name()
	model.Register(name, func(opts ...model.Option) model.Model {
		return &turnTestModel{opts: model.NewOptions(opts...), turn: func(_ context.Context, r *model.Request) (*model.Response, error) {
			if r.Continuation == nil {
				return &model.Response{ToolCalls: []model.ToolCall{{ID: "one", Name: "effect"}}, Continuation: &model.Continuation{Provider: name}}, nil
			}
			followups++
			if followups == 1 {
				return nil, &model.HTTPError{Code: 503, Status: "unavailable"}
			}
			return &model.Response{Reply: "done"}, nil
		}}
	})
	a := New(Provider(name), WithRegistry(registry.NewMemoryRegistry()), WithStore(store.NewMemoryStore()), ModelRetry(2, time.Millisecond), WithTool("effect", "effect", nil, func(context.Context, map[string]any) (string, error) { effects++; return "done", nil }))
	response, err := a.Ask(context.Background(), "work")
	if err != nil {
		t.Fatal(err)
	}
	if effects != 1 || followups != 2 || response.Reply != "done" {
		t.Fatalf("effects=%d followups=%d response=%+v", effects, followups, response)
	}
}

func TestStrictAgentResumeKeepsTurnAndToolResults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.db")
	cp, err := flow.OpenJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	name := t.Name()
	effects, requests := 0, 0
	model.Register(name, func(opts ...model.Option) model.Model {
		return &turnTestModel{opts: model.NewOptions(opts...), turn: func(_ context.Context, r *model.Request) (*model.Response, error) {
			requests++
			return &model.Response{Reply: "recovered"}, nil
		}}
	})
	call := model.ToolCall{ID: "one", Name: "effect", Input: map[string]any{}}
	saved := flow.Run{ID: "stable", Flow: "recover", Status: "running", State: flow.State{Stage: agentAskStep, Data: []byte("work")}, ToolSteps: 1, ModelTurns: 1, PendingTurn: &model.Response{ToolCalls: []model.ToolCall{call}, Continuation: &model.Continuation{Provider: name}}, Steps: []flow.StepRecord{{Name: agentAskStep, Status: "in_progress"}, {Name: toolCheckpointName(call), Status: "done", Result: "saved result"}}}
	if err := cp.Save(context.Background(), saved); err != nil {
		t.Fatal(err)
	}
	if err := cp.Close(); err != nil {
		t.Fatal(err)
	}
	cp, err = flow.OpenJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	defer cp.Close()
	a := New(Name("recover"), Provider(name), WithRegistry(registry.NewMemoryRegistry()), WithStore(store.NewMemoryStore()), WithCheckpoint(cp), StrictRecovery(), MaxSteps(1), WithTool("effect", "effect", nil, func(context.Context, map[string]any) (string, error) { effects++; return "wrong", nil }))
	response, err := Run(context.Background(), a, "stable", "work")
	if err != nil {
		t.Fatal(err)
	}
	if effects != 0 || requests != 1 || response.Reply != "recovered" {
		t.Fatalf("effects=%d requests=%d response=%+v", effects, requests, response)
	}
	if _, err := Run(context.Background(), a, "stable", "work"); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatal("completed run repeated model call")
	}
}

func TestStrictAgentModelBudgetSurvivesRestart(t *testing.T) {
	cp, err := flow.OpenJournal(filepath.Join(t.TempDir(), "runs.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer cp.Close()
	name := t.Name()
	requests := 0
	model.Register(name, func(opts ...model.Option) model.Model {
		return &turnTestModel{opts: model.NewOptions(opts...), turn: func(context.Context, *model.Request) (*model.Response, error) {
			requests++
			return nil, &model.HTTPError{Code: 503, Status: "unavailable"}
		}}
	})
	makeAgent := func() Agent {
		return New(Name("budget"), Provider(name), WithRegistry(registry.NewMemoryRegistry()), WithStore(store.NewMemoryStore()), WithCheckpoint(cp), StrictRecovery())
	}
	if _, err := Run(context.Background(), makeAgent(), "stable", "work"); err == nil {
		t.Fatal("expected provider failure")
	}
	if _, err := Run(context.Background(), makeAgent(), "stable", "work"); !errors.Is(err, model.ErrLimit) {
		t.Fatalf("expected consumed budget: %v", err)
	}
	if requests != 1 {
		t.Fatalf("replenished budget: %d", requests)
	}
}

func TestStableAgentChatDeduplicatesRPC(t *testing.T) {
	cp, err := flow.OpenJournal(filepath.Join(t.TempDir(), "runs.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer cp.Close()
	requests := 0
	name := t.Name()
	model.Register(name, func(opts ...model.Option) model.Model {
		return &turnTestModel{opts: model.NewOptions(opts...), turn: func(context.Context, *model.Request) (*model.Response, error) {
			requests++
			return &model.Response{Reply: "done"}, nil
		}}
	})
	a := New(Name("rpc"), Provider(name), WithRegistry(registry.NewMemoryRegistry()), WithStore(store.NewMemoryStore()), WithCheckpoint(cp), StrictRecovery()).(*agentImpl)
	ctx := metadata.NewContext(context.Background(), metadata.Metadata{"micro-agent-run-id": "stable-rpc"})
	for i := 0; i < 2; i++ {
		var rsp pb.ChatResponse
		if err := a.Chat(ctx, &pb.ChatRequest{Message: "work", ParentId: "parent"}, &rsp); err != nil {
			t.Fatal(err)
		}
		if rsp.RunId != "stable-rpc" || rsp.ParentId != "parent" {
			t.Fatalf("lost lineage: %+v", &rsp)
		}
	}
	if requests != 1 {
		t.Fatalf("replayed model %d times", requests)
	}
}
