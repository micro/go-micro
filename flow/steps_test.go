package flow

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-micro.dev/v6/ai"
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

func TestFlowStepContextIncludesRunInfo(t *testing.T) {
	var got ai.RunInfo
	step := Step{Name: "inspect", Run: func(ctx context.Context, in State) (State, error) {
		var ok bool
		got, ok = ai.RunInfoFrom(ctx)
		if !ok {
			t.Fatal("RunInfo missing from step context")
		}
		in.Data = []byte("ok")
		return in, nil
	}}

	mem := store.NewMemoryStore()
	f := New("correlated",
		WithCheckpoint(StoreCheckpoint(mem, "correlated")),
		Steps(step),
	)
	ctx := ai.WithRunInfo(context.Background(), ai.RunInfo{RunID: "agent-run-1", Agent: "planner"})
	if err := f.Execute(ctx, "start"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got.Agent != "correlated" {
		t.Fatalf("RunInfo.Agent = %q, want correlated", got.Agent)
	}
	if got.RunID == "" {
		t.Fatal("RunInfo.RunID is empty")
	}
	if got.Flow != "correlated" {
		t.Fatalf("RunInfo.Flow = %q, want correlated", got.Flow)
	}
	if got.ParentID != "agent-run-1" {
		t.Fatalf("RunInfo.ParentID = %q, want agent-run-1", got.ParentID)
	}
	if got.Step != "inspect" {
		t.Fatalf("RunInfo.Step = %q, want inspect", got.Step)
	}
	runs, err := StoreCheckpoint(mem, "correlated").List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(runs) != 1 || runs[0].ParentID != "agent-run-1" {
		t.Fatalf("persisted parent id = %+v, want agent-run-1", runs)
	}
}

func TestFlowResumePendingResumesOldestRunsUntilFailure(t *testing.T) {
	mem := store.NewMemoryStore()
	ctx := context.Background()
	var calls int
	step := Step{Name: "work", Run: func(_ context.Context, in State) (State, error) {
		calls++
		if in.String() == "block" {
			return in, errors.New("still blocked")
		}
		in.Data = []byte(in.String() + "-done")
		return in, nil
	}}
	f := New("resume-pending", WithCheckpoint(StoreCheckpoint(mem, "resume-pending")), Steps(step))

	base := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	runs := []Run{
		{
			ID:      "run-ok",
			Flow:    "resume-pending",
			State:   State{Stage: "work", Data: []byte("ok")},
			Steps:   []StepRecord{{Name: "work", Status: "failed", Error: "temporary"}},
			Status:  "failed",
			Started: base,
		},
		{
			ID:      "run-blocked",
			Flow:    "resume-pending",
			State:   State{Stage: "work", Data: []byte("block")},
			Steps:   []StepRecord{{Name: "work", Status: "failed", Error: "temporary"}},
			Status:  "failed",
			Started: base.Add(time.Minute),
		},
		{
			ID:      "run-later",
			Flow:    "resume-pending",
			State:   State{Stage: "work", Data: []byte("later")},
			Steps:   []StepRecord{{Name: "work", Status: "failed", Error: "temporary"}},
			Status:  "failed",
			Started: base.Add(2 * time.Minute),
		},
	}
	for _, run := range runs {
		if err := f.checkpoint.Save(ctx, run); err != nil {
			t.Fatalf("Save(%s): %v", run.ID, err)
		}
	}

	failedRun, err := f.ResumePending(ctx)
	if err == nil {
		t.Fatal("expected ResumePending to stop at the blocked run")
	}
	if failedRun != "run-blocked" {
		t.Fatalf("failed run = %q, want run-blocked", failedRun)
	}
	if calls != 2 {
		t.Fatalf("ResumePending should stop before later runs; got %d calls", calls)
	}

	run, ok, err := f.checkpoint.Load(ctx, "run-ok")
	if err != nil || !ok {
		t.Fatalf("Load(run-ok) ok=%v err=%v", ok, err)
	}
	if run.Status != "done" || run.State.String() != "ok-done" {
		t.Fatalf("run-ok not resumed successfully: %+v", run)
	}
	run, ok, err = f.checkpoint.Load(ctx, "run-later")
	if err != nil || !ok {
		t.Fatalf("Load(run-later) ok=%v err=%v", ok, err)
	}
	if run.Status != "failed" {
		t.Fatalf("run-later should not be resumed after a failure, got %+v", run)
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

func TestFlowStepRetryBackoffWaitsBetweenAttempts(t *testing.T) {
	var attempts int
	step := Step{Name: "transient", Run: func(_ context.Context, in State) (State, error) {
		attempts++
		if attempts == 1 {
			return in, errors.New("transient")
		}
		in.Data = []byte("ok")
		return in, nil
	}}

	f := New("retry-backoff",
		WithCheckpoint(StoreCheckpoint(store.NewMemoryStore(), "retry-backoff")),
		Retry(1),
		RetryBackoff(10*time.Millisecond),
		Steps(step),
	)
	start := time.Now()
	if err := f.Execute(context.Background(), ""); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("want 2 attempts, got %d", attempts)
	}
	if elapsed := time.Since(start); elapsed < 10*time.Millisecond {
		t.Fatalf("retry backoff was not observed; elapsed %s", elapsed)
	}
}

func TestFlowStepRetryBackoffStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var attempts int
	step := Step{Name: "cancelbackoff", Run: func(_ context.Context, in State) (State, error) {
		attempts++
		cancel()
		return in, errors.New("transient")
	}}

	f := New("cancel-backoff",
		WithCheckpoint(StoreCheckpoint(store.NewMemoryStore(), "cancel-backoff")),
		Retry(1),
		RetryBackoff(time.Hour),
		Steps(step),
	)
	err := f.Execute(ctx, "")
	if err == nil {
		t.Fatal("expected the canceled run to fail")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("want a context.Canceled error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("cancellation should stop during backoff before retrying, got %d attempts", attempts)
	}
}

// A canceled run stops retrying immediately instead of burning the whole
// retry budget, and surfaces the context error.
func TestFlowStepRetryStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var attempts int
	step := Step{Name: "cancelaware", Run: func(_ context.Context, in State) (State, error) {
		attempts++
		cancel() // the run is canceled while this step is in flight
		return in, errors.New("transient")
	}}

	f := New("cancelretry",
		WithCheckpoint(StoreCheckpoint(store.NewMemoryStore(), "cancelretry")),
		Retry(5), // would be 6 tries without the cancellation check
		Steps(step),
	)

	err := f.Execute(ctx, "")
	if err == nil {
		t.Fatal("expected the canceled run to fail")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("want a context.Canceled error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("cancellation should stop retries after the first attempt, got %d", attempts)
	}
}

func TestFlowStepNamesMustBeUnique(t *testing.T) {
	step := Step{Name: "work", Run: func(_ context.Context, in State) (State, error) {
		return in, nil
	}}
	f := New("duplicate-steps",
		WithCheckpoint(StoreCheckpoint(store.NewMemoryStore(), "duplicate-steps")),
		Steps(step, step),
	)

	err := f.Execute(context.Background(), "")
	if err == nil {
		t.Fatal("expected duplicate step names to fail validation")
	}
	if got, want := err.Error(), `flow: duplicate step name "work"`; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestFlowStepNamesMustBeNonEmpty(t *testing.T) {
	f := New("empty-step-name",
		WithCheckpoint(StoreCheckpoint(store.NewMemoryStore(), "empty-step-name")),
		Steps(Step{Name: "", Run: func(_ context.Context, in State) (State, error) {
			return in, nil
		}}),
	)

	err := f.Execute(context.Background(), "")
	if err == nil {
		t.Fatal("expected an empty step name to fail validation")
	}
	if got, want := err.Error(), "flow: step 0 has an empty name"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

// A step with no Run function is reported as a configuration error rather
// than panicking the run.
func TestFlowStepNilRun(t *testing.T) {
	f := New("nilstep",
		WithCheckpoint(StoreCheckpoint(store.NewMemoryStore(), "nilstep")),
		Steps(Step{Name: "missing"}),
	)
	err := f.Execute(context.Background(), "")
	if err == nil {
		t.Fatal("expected an error for a step with no Run function")
	}
}

func TestStoreCheckpointListReturnsRunsInStartedOrder(t *testing.T) {
	cp := StoreCheckpoint(store.NewMemoryStore(), "ordered")
	ctx := context.Background()
	base := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	runs := []Run{
		{ID: "run-c", Flow: "ordered", Status: "failed", Started: base.Add(2 * time.Minute)},
		{ID: "run-a", Flow: "ordered", Status: "failed", Started: base},
		{ID: "run-b", Flow: "ordered", Status: "failed", Started: base.Add(time.Minute)},
	}
	for _, run := range runs {
		if err := cp.Save(ctx, run); err != nil {
			t.Fatalf("Save(%s): %v", run.ID, err)
		}
	}

	got, err := cp.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("List returned %d runs, want 3", len(got))
	}
	want := []string{"run-a", "run-b", "run-c"}
	for i, id := range want {
		if got[i].ID != id {
			t.Fatalf("run %d = %q, want %q (all runs: %+v)", i, got[i].ID, id, got)
		}
	}
}

func TestStoreCheckpointHonorsCanceledContext(t *testing.T) {
	cp := StoreCheckpoint(store.NewMemoryStore(), "canceled")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	run := Run{ID: "canceled", Started: time.Now()}

	if err := cp.Save(ctx, run); !errors.Is(err, context.Canceled) {
		t.Fatalf("Save error = %v, want context.Canceled", err)
	}
	if _, ok, err := cp.Load(ctx, run.ID); !errors.Is(err, context.Canceled) || ok {
		t.Fatalf("Load ok, error = %v, %v; want false, context.Canceled", ok, err)
	}
	if err := cp.Delete(ctx, run.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("Delete error = %v, want context.Canceled", err)
	}
	if _, err := cp.List(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("List error = %v, want context.Canceled", err)
	}
}

type failingCheckpoint struct {
	err error
}

func (c failingCheckpoint) Save(context.Context, Run) error { return c.err }
func (c failingCheckpoint) Load(context.Context, string) (Run, bool, error) {
	return Run{}, false, c.err
}
func (c failingCheckpoint) Delete(context.Context, string) error { return c.err }
func (c failingCheckpoint) List(context.Context) ([]Run, error)  { return nil, c.err }

func TestFlowCheckpointSaveFailureStopsRun(t *testing.T) {
	checkpointErr := errors.New("checkpoint unavailable")
	var ran bool
	f := New("checkpoint-fails",
		WithCheckpoint(failingCheckpoint{err: checkpointErr}),
		Steps(Step{Name: "work", Run: func(_ context.Context, in State) (State, error) {
			ran = true
			return in, nil
		}}),
	)

	err := f.Execute(context.Background(), "start")
	if !errors.Is(err, checkpointErr) {
		t.Fatalf("Execute error = %v, want checkpoint error", err)
	}
	if ran {
		t.Fatal("step ran even though the in-progress checkpoint failed")
	}
}

func TestFlowDeleteOnSuccessFailureIsReturned(t *testing.T) {
	checkpointErr := errors.New("delete unavailable")
	cp := &deleteFailCheckpoint{Checkpoint: StoreCheckpoint(store.NewMemoryStore(), "delete-fails"), err: checkpointErr}
	f := New("delete-fails",
		WithCheckpoint(cp),
		DeleteOnSuccess(),
		Steps(appendStep("work")),
	)

	err := f.Execute(context.Background(), "")
	if !errors.Is(err, checkpointErr) {
		t.Fatalf("Execute error = %v, want delete error", err)
	}
}

type deleteFailCheckpoint struct {
	Checkpoint
	err error
}

func (c *deleteFailCheckpoint) Delete(context.Context, string) error { return c.err }

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
