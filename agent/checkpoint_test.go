package agent

import (
	"context"
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
