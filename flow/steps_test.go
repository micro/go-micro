package flow

import (
	"context"
	"errors"
	"testing"

	"go-micro.dev/v6/store"
)

// appendStep returns a step that appends its name to the carried data,
// so a run's path is visible in the final State.
func appendStep(name string) Step {
	return Step{Name: name, Run: func(_ context.Context, in State) (State, error) {
		s := in.String()
		if s != "" {
			s += ","
		}
		in.Data = []byte(s + name)
		return in, nil
	}}
}

func TestFlowStepsRunInOrder(t *testing.T) {
	f := New("seq",
		WithCheckpoint(StoreCheckpoint(store.NewMemoryStore(), "seq")),
		Steps(appendStep("a"), appendStep("b"), appendStep("c")),
	)
	if err := f.Execute(context.Background(), ""); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	res := f.Results()
	if len(res) != 1 || res[0].Answer != "a,b,c" {
		t.Fatalf("steps ran out of order: %+v", res)
	}
}

// A run that fails mid-way is persisted at the failing step and resumes
// there — without re-running the completed steps.
func TestFlowCheckpointResume(t *testing.T) {
	mem := store.NewMemoryStore()
	var firstCalls, fixed int

	steps := []Step{
		{Name: "first", Run: func(_ context.Context, in State) (State, error) {
			firstCalls++
			in.Data = []byte("first-done")
			return in, nil
		}},
		{Name: "flaky", Run: func(_ context.Context, in State) (State, error) {
			if fixed == 0 {
				return in, errors.New("dependency unavailable")
			}
			in.Data = []byte("flaky-done")
			return in, nil
		}},
	}

	f := New("resumable", WithCheckpoint(StoreCheckpoint(mem, "resumable")), Steps(steps...))

	// First run fails at "flaky".
	if err := f.Execute(context.Background(), "start"); err == nil {
		t.Fatal("expected the run to fail at the flaky step")
	}

	pend, _ := f.Pending(context.Background())
	if len(pend) != 1 {
		t.Fatalf("expected 1 pending run, got %d", len(pend))
	}
	if pend[0].State.Stage != "flaky" {
		t.Fatalf("run should be checkpointed at the flaky step, got stage %q", pend[0].State.Stage)
	}
	runID := pend[0].ID

	// The dependency recovers; resume continues from where it stopped.
	fixed = 1
	if err := f.Resume(context.Background(), runID); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if firstCalls != 1 {
		t.Errorf("completed step should not re-run on resume; first called %d times", firstCalls)
	}
	if pend, _ := f.Pending(context.Background()); len(pend) != 0 {
		t.Errorf("expected no pending runs after a successful resume, got %d", len(pend))
	}
}

// A flow-level Retry re-runs a failing step until it succeeds.
func TestFlowStepRetry(t *testing.T) {
	var attempts int
	step := Step{Name: "transient", Run: func(_ context.Context, in State) (State, error) {
		attempts++
		if attempts < 3 {
			return in, errors.New("transient")
		}
		in.Data = []byte("ok")
		return in, nil
	}}

	f := New("retrying",
		WithCheckpoint(StoreCheckpoint(store.NewMemoryStore(), "retrying")),
		Retry(2), // up to 3 tries
		Steps(step),
	)
	if err := f.Execute(context.Background(), ""); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if attempts != 3 {
		t.Errorf("want 3 attempts with Retry(2), got %d", attempts)
	}
}

// A per-step Retry overrides the flow default.
func TestFlowStepRetryOverride(t *testing.T) {
	var attempts int
	step := Step{Name: "capped", Retry: 1, Run: func(_ context.Context, in State) (State, error) {
		attempts++
		return in, errors.New("always fails")
	}}

	f := New("override",
		WithCheckpoint(StoreCheckpoint(store.NewMemoryStore(), "override")),
		Retry(5), // would be 6 tries; the step's Retry:1 caps it at 2
		Steps(step),
	)
	_ = f.Execute(context.Background(), "")
	if attempts != 2 {
		t.Errorf("per-step Retry(1) should cap tries at 2, got %d", attempts)
	}
}

func TestStateSetScan(t *testing.T) {
	var s State
	type payload struct {
		Email string `json:"email"`
	}
	if err := s.Set(payload{Email: "a@b.com"}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	var got payload
	if err := s.Scan(&got); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if got.Email != "a@b.com" {
		t.Errorf("round-trip failed: %+v", got)
	}
}

func TestFlowStepStopsRetryingWhenContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var attempts int
	step := Step{Name: "cancel", Run: func(_ context.Context, in State) (State, error) {
		attempts++
		cancel()
		return in, errors.New("transient")
	}}

	f := New("cancel-retry",
		WithCheckpoint(StoreCheckpoint(store.NewMemoryStore(), "cancel-retry")),
		Retry(5),
		Steps(step),
	)
	if err := f.Execute(ctx, ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("Execute error = %v, want context.Canceled", err)
	}
	if attempts != 1 {
		t.Errorf("canceled flow should not retry; got %d attempts", attempts)
	}
}

func TestFlowStepRequiresRunFunc(t *testing.T) {
	f := New("nil-step",
		WithCheckpoint(StoreCheckpoint(store.NewMemoryStore(), "nil-step")),
		Steps(Step{Name: "missing"}),
	)
	if err := f.Execute(context.Background(), ""); err == nil {
		t.Fatal("expected missing Run function error")
	}
}
