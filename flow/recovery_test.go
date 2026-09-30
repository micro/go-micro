package flow

import (
	"context"
	"errors"
	"go-micro.dev/v6/model"
	"path/filepath"
	"sync"
	"testing"
)

func TestStrictRecoveryRestartAndAdmission(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "runs.db")
	cp, err := OpenCheckpoint(path)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	steps := []Step{{Name: "effect", Idempotent: true, Run: func(_ context.Context, in State) (State, error) { count++; return in, nil }}, AwaitStep("approve", "approval", "Continue?"), {Name: "finish", Idempotent: true, Run: func(_ context.Context, in State) (State, error) { count++; return in, nil }}}
	f := New("recovery", StrictRecovery(), WithCheckpoint(cp), Steps(steps...))
	run, err := f.Start(ctx, "stable", "input")
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "waiting" || count != 1 {
		t.Fatalf("run=%+v count=%d", run, count)
	}
	if err := cp.Close(); err != nil {
		t.Fatal(err)
	}
	cp, err = OpenCheckpoint(path)
	if err != nil {
		t.Fatal(err)
	}
	defer cp.Close()
	f = New("recovery", StrictRecovery(), WithCheckpoint(cp), Steps(steps...))
	if err := f.Resume(ctx, "stable"); err == nil {
		t.Fatal("waiting run resumed without approval")
	}
	if err := f.ResumeWith(ctx, "stable", "approved"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := f.Start(ctx, "stable", "duplicate"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if count != 2 {
		t.Fatalf("replayed effects: %d", count)
	}
}

func TestStrictRecoveryBudgetAndAmbiguousEffects(t *testing.T) {
	for _, ambiguous := range []bool{false, true} {
		t.Run(map[bool]string{false: "budget", true: "ambiguous"}[ambiguous], func(t *testing.T) {
			cp, err := OpenCheckpoint(filepath.Join(t.TempDir(), "runs.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer cp.Close()
			called := false
			f := New("bounded", StrictRecovery(), WithCheckpoint(cp), Steps(Step{Name: "effect", Run: func(_ context.Context, in State) (State, error) { called = true; return in, nil }}))
			status := "failed"
			if ambiguous {
				status = "in_progress"
			}
			r := Run{ID: "one", Flow: "bounded", State: State{Stage: "effect"}, Status: "running", Steps: []StepRecord{{Name: "effect", Attempts: 1, Status: status}}}
			if err := cp.Save(context.Background(), r); err != nil {
				t.Fatal(err)
			}
			err = f.Resume(context.Background(), r.ID)
			want := ErrLimit
			if ambiguous {
				want = ErrAmbiguous
			}
			if !errors.Is(err, want) || called {
				t.Fatalf("err=%v called=%v", err, called)
			}
		})
	}
}

func TestStrictRecoveryCancellationIsTerminal(t *testing.T) {
	cp, err := OpenCheckpoint(filepath.Join(t.TempDir(), "runs.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer cp.Close()
	ctx, cancel := context.WithCancel(context.Background())
	f := New("cancel", StrictRecovery(), WithCheckpoint(cp), Steps(Step{Name: "work", Run: func(_ context.Context, in State) (State, error) { cancel(); return in, context.Canceled }}))
	_, err = f.Start(ctx, "canceled", "input")
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	r, _, err := cp.Load(context.Background(), "canceled")
	if err != nil || r.Status != "canceled" {
		t.Fatalf("record=%+v err=%v", r, err)
	}
	if err := f.Resume(context.Background(), r.ID); err == nil {
		t.Fatal("canceled run resumed")
	}
}

func TestJournalRejectsConcurrentOwner(t *testing.T) {
	// The on-disk process lock is separate from per-run goroutine ownership.
	path := filepath.Join(t.TempDir(), "runs.db")
	cp, err := OpenCheckpoint(path)
	if err != nil {
		t.Fatal(err)
	}
	defer cp.Close()
	other, err := OpenCheckpoint(path)
	if err == nil {
		_ = other.Close()
		t.Fatal("second journal owner admitted")
	}
	if model.ClassifyError(ErrLimit) != model.ErrorKindExhausted {
		t.Fatal("limit outcome lost")
	}
}

type failAfterEffect struct {
	Checkpoint
	locker interface {
		Lock(context.Context, string) (func(), error)
	}
	fail bool
}

func (c *failAfterEffect) Lock(ctx context.Context, id string) (func(), error) {
	return c.locker.Lock(ctx, id)
}
func (c *failAfterEffect) Save(ctx context.Context, r Run) error {
	if c.fail && len(r.Steps) > 0 && r.Steps[0].Status == "done" {
		c.fail = false
		return errors.New("lost checkpoint after effect")
	}
	return c.Checkpoint.Save(ctx, r)
}
func TestStrictRecoveryEffectBeforeCheckpoint(t *testing.T) {
	j, err := OpenCheckpoint(filepath.Join(t.TempDir(), "runs.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	cp := &failAfterEffect{Checkpoint: j, locker: j, fail: true}
	effects := map[string]bool{}
	calls := 0
	step := Step{Name: "effect", Idempotent: true, Run: func(ctx context.Context, in State) (State, error) {
		calls++
		effects[OperationKey(ctx)] = true
		return in, nil
	}}
	f := New("effect", StrictRecovery(), Retry(1), WithCheckpoint(cp), Steps(step))
	if _, err := f.Start(context.Background(), "stable", "input"); err == nil {
		t.Fatal("checkpoint failure hidden")
	}
	// A new flow sees the admitted but unrecorded outcome and reuses the same key.
	f = New("effect", StrictRecovery(), Retry(1), WithCheckpoint(j), Steps(step))
	if err := f.Resume(context.Background(), "stable"); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(effects) != 1 {
		t.Fatalf("calls=%d effects=%d", calls, len(effects))
	}
}
