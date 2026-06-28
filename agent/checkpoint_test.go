package agent

import (
	"context"
	"errors"
	"testing"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/flow"
	"go-micro.dev/v6/store"
)

func TestResumeCompletedCheckpointDoesNotReplayModel(t *testing.T) {
	ctx := context.Background()
	cp := flow.StoreCheckpoint(store.NewStore(), "durable-agent")
	calls := 0
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		calls++
		return &ai.Response{Reply: "done"}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("durable-agent"), WithCheckpoint(cp))
	resp, err := a.Ask(ctx, "finish the work")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if calls != 1 {
		t.Fatalf("model calls after Ask = %d, want 1", calls)
	}

	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		calls++
		t.Fatal("Resume of a completed run replayed the model")
		return nil, nil
	}
	resumed, err := Resume(ctx, a, resp.RunID)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if resumed.Reply != "done" {
		t.Fatalf("resumed reply = %q, want done", resumed.Reply)
	}
	if resumed.RunID != resp.RunID {
		t.Fatalf("resumed run id = %q, want %q", resumed.RunID, resp.RunID)
	}
	if calls != 1 {
		t.Fatalf("model calls after Resume = %d, want 1", calls)
	}
}

func TestResumeFailedCheckpointDoesNotReplayCompletedTool(t *testing.T) {
	ctx := context.Background()
	cp := flow.StoreCheckpoint(store.NewStore(), "tool-resume-agent")
	toolRuns := 0
	first := true
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		if opts.ToolHandler != nil {
			res := opts.ToolHandler(ctx, ai.ToolCall{ID: "call-1", Name: "external.charge", Input: map[string]any{"order": "42"}})
			if res.Content != "charged" {
				t.Fatalf("tool result = %q, want charged", res.Content)
			}
		}
		if first {
			first = false
			return nil, errors.New("model connection dropped after tool")
		}
		return &ai.Response{Reply: "finished from checkpoint"}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("tool-resume-agent"), WithCheckpoint(cp),
		WithTool("external.charge", "charge once", nil, func(context.Context, map[string]any) (string, error) {
			toolRuns++
			return "charged", nil
		}))
	_, err := a.Ask(ctx, "charge order 42")
	if err == nil {
		t.Fatal("Ask succeeded, want simulated failure")
	}
	if toolRuns != 1 {
		t.Fatalf("tool executions after failed Ask = %d, want 1", toolRuns)
	}

	runs, err := Pending(ctx, a)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("Pending returned %d runs, want 1", len(runs))
	}
	resp, err := Resume(ctx, a, runs[0].ID)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if resp.Reply != "finished from checkpoint" {
		t.Fatalf("Resume reply = %q", resp.Reply)
	}
	if toolRuns != 1 {
		t.Fatalf("tool executions after Resume = %d, want completed tool was not replayed", toolRuns)
	}
}

func TestPendingReturnsUnfinishedAgentRuns(t *testing.T) {
	ctx := context.Background()
	cp := flow.StoreCheckpoint(store.NewStore(), "pending-agent")
	run := flow.Run{ID: "run-1", Flow: "pending-agent", Status: "failed", State: flow.State{Stage: agentAskStep, Data: []byte("retry me")}}
	if err := cp.Save(ctx, run); err != nil {
		t.Fatalf("Save: %v", err)
	}
	a := newTestAgent(Name("pending-agent"), WithCheckpoint(cp))
	runs, err := Pending(ctx, a)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(runs) != 1 || runs[0].ID != "run-1" {
		t.Fatalf("Pending = %#v, want run-1", runs)
	}
}

func TestApprovalDenialPausesCheckpointedRunAndResumeContinues(t *testing.T) {
	ctx := context.Background()
	cp := flow.StoreCheckpoint(store.NewStore(), "approval-agent")
	calls := 0
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		calls++
		if opts.ToolHandler != nil {
			opts.ToolHandler(ctx, ai.ToolCall{ID: "call-1", Name: "external.approve", Input: map[string]any{"id": "42"}})
		}
		return &ai.Response{Reply: "model saw approval result"}, nil
	}
	defer func() { fakeGen = nil }()

	approved := false
	a := newTestAgent(Name("approval-agent"), WithCheckpoint(cp),
		WithTool("external.approve", "guarded external action", nil, func(context.Context, map[string]any) (string, error) { return "ok", nil }),
		ApproveTool(func(tool string, input map[string]any) (bool, string) {
			return approved, "waiting for operator"
		}))
	_, err := a.Ask(ctx, "send the guarded update")
	if err == nil {
		t.Fatal("Ask succeeded, want paused approval error")
	}

	runs, err := Pending(ctx, a)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("Pending returned %d runs, want 1: %#v", len(runs), runs)
	}
	if runs[0].Status != "paused" || runs[0].State.Stage != agentApprovalStep {
		t.Fatalf("run status/stage = %s/%s, want paused/%s", runs[0].Status, runs[0].State.Stage, agentApprovalStep)
	}
	if got := string(runs[0].State.Data); got != "send the guarded update" {
		t.Fatalf("paused run data = %q", got)
	}

	approved = true
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		calls++
		if opts.ToolHandler != nil {
			res := opts.ToolHandler(ctx, ai.ToolCall{ID: "call-2", Name: "external.approve", Input: map[string]any{"id": "42"}})
			if res.Refused != "" {
				t.Fatalf("resumed call was refused: %#v", res)
			}
		}
		return &ai.Response{Reply: "done after approval"}, nil
	}
	resp, err := Resume(ctx, a, runs[0].ID)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if resp.Reply != "done after approval" {
		t.Fatalf("Resume reply = %q", resp.Reply)
	}
	loaded, ok, err := cp.Load(ctx, runs[0].ID)
	if err != nil || !ok {
		t.Fatalf("Load resumed run ok=%v err=%v", ok, err)
	}
	if loaded.Status != "done" {
		t.Fatalf("resumed run status = %q, want done", loaded.Status)
	}
	if calls != 2 {
		t.Fatalf("model calls = %d, want 2", calls)
	}
}
