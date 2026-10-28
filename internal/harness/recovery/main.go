// Command recovery is the provider-free restart and failure-injection release
// gate for durable flows. It deliberately replaces the Flow value between
// attempts while retaining the checkpoint, which models a host restart.
package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"go-micro.dev/v6/flow"
	"go-micro.dev/v6/model"
)

func main() {
	if err := run(context.Background()); err != nil {
		panic(err)
	}
	fmt.Println("recovery fixture: PASS")
}

func run(ctx context.Context) error {
	cp := newCheckpoint()
	effects := newEffects()
	var reserveCalls, chargeCalls, finalCalls int
	var callsMu sync.Mutex
	crashAfterCharge := true

	steps := func(concurrentFinal bool) []flow.Step {
		return []flow.Step{
			{Name: "reserve", Run: func(_ context.Context, in flow.State) (flow.State, error) {
				callsMu.Lock()
				reserveCalls++
				callsMu.Unlock()
				effects.apply("reserve:order-42")
				return in, nil
			}},
			{Name: "charge", Run: func(_ context.Context, in flow.State) (flow.State, error) {
				callsMu.Lock()
				chargeCalls++
				callsMu.Unlock()
				effects.apply("charge:order-42")
				if crashAfterCharge {
					crashAfterCharge = false
					return in, errors.New("injected crash after external charge, before checkpoint")
				}
				return in, nil
			}},
			{Name: "approval", Run: flow.Await("ship", "approve shipment")},
			{Name: "finalize", Run: func(_ context.Context, in flow.State) (flow.State, error) {
				callsMu.Lock()
				finalCalls++
				calls := finalCalls
				callsMu.Unlock()
				effects.apply("finalize:order-42")
				if concurrentFinal {
					finalBarrier.arrive(calls)
				}
				return in, nil
			}},
		}
	}

	root := model.WithRunInfo(ctx, model.RunInfo{RunID: "request-7", Dispatch: "direct"})
	host1 := flow.New("recovery-gate", flow.Steps(steps(false)...), flow.WithCheckpoint(cp))
	if err := host1.Execute(root, `{"order":"42"}`); err == nil {
		return errors.New("injected ambiguous charge did not fail")
	}
	runs, _ := cp.List(ctx)
	if len(runs) != 1 || runs[0].State.Stage != "charge" || runs[0].ParentID != "request-7" {
		return fmt.Errorf("failed checkpoint or lineage was not preserved: %+v", runs)
	}
	runID := runs[0].ID

	// A new Flow value represents a replacement host using the durable backend.
	// The charge is reconciled by its idempotency key when the ambiguous step is
	// retried; the completed reserve step must not run again.
	host2 := flow.New("recovery-gate", flow.Steps(steps(false)...), flow.WithCheckpoint(cp))
	if err := host2.Resume(ctx, runID); err != nil {
		return fmt.Errorf("restart resume: %w", err)
	}
	waiting, _ := host2.Waiting(ctx)
	pending, _ := host2.Pending(ctx)
	if len(waiting) != 1 || len(pending) != 0 || waiting[0].ParentID != "request-7" {
		return fmt.Errorf("waiting run was not kept paused with its lineage")
	}
	if reserveCalls != 1 || chargeCalls != 2 || effects.count("charge:order-42") != 1 {
		return fmt.Errorf("restart replay contract failed: reserve=%d charge calls=%d effects=%d", reserveCalls, chargeCalls, effects.count("charge:order-42"))
	}

	// Two deliveries race to approve the same saved run. The checkpoint API does
	// not promise a cross-host lease in v6, so both may reach the operation. The
	// stable business idempotency key is the required safety boundary.
	finalBarrier = newBarrier(2)
	host3 := flow.New("recovery-gate", flow.Steps(steps(true)...), flow.WithCheckpoint(cp))
	errCh := make(chan error, 2)
	for range 2 {
		go func() { errCh <- host3.ResumeWith(ctx, runID, "approved") }()
	}
	for range 2 {
		if err := <-errCh; err != nil {
			return fmt.Errorf("concurrent duplicate resume: %w", err)
		}
	}
	completed, ok, _ := cp.Load(ctx, runID)
	if !ok || completed.Status != "done" || completed.ParentID != "request-7" {
		return fmt.Errorf("completed run lost status or lineage: %+v", completed)
	}
	if finalCalls != 2 || effects.count("finalize:order-42") != 1 {
		return fmt.Errorf("idempotent final operation duplicated: calls=%d effects=%d", finalCalls, effects.count("finalize:order-42"))
	}

	// Exhaustion is a failed outcome, never a successful completion.
	exhaustCP := newCheckpoint()
	attempts := 0
	exhaust := flow.New("exhaustion-gate",
		flow.Steps(flow.Step{Name: "bounded", Run: func(_ context.Context, in flow.State) (flow.State, error) {
			attempts++
			return in, errors.New("dependency unavailable")
		}}),
		flow.Retry(1), flow.WithCheckpoint(exhaustCP),
	)
	if err := exhaust.Execute(ctx, "work"); err == nil {
		return errors.New("exhausted retry budget was reported as success")
	}
	exhausted, _ := exhaustCP.List(ctx)
	if attempts != 2 || len(exhausted) != 1 || exhausted[0].Status != "failed" || exhausted[0].Steps[0].Attempts != 2 {
		return fmt.Errorf("retry exhaustion was not recorded: attempts=%d runs=%+v", attempts, exhausted)
	}
	return nil
}

type memoryCheckpoint struct {
	mu   sync.Mutex
	runs map[string]flow.Run
}

func newCheckpoint() *memoryCheckpoint { return &memoryCheckpoint{runs: make(map[string]flow.Run)} }

func (c *memoryCheckpoint) Save(_ context.Context, run flow.Run) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.runs[run.ID] = cloneRun(run)
	return nil
}

func (c *memoryCheckpoint) Load(_ context.Context, id string) (flow.Run, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	run, ok := c.runs[id]
	return cloneRun(run), ok, nil
}

func (c *memoryCheckpoint) Delete(_ context.Context, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.runs, id)
	return nil
}

func (c *memoryCheckpoint) List(_ context.Context) ([]flow.Run, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	runs := make([]flow.Run, 0, len(c.runs))
	for _, run := range c.runs {
		runs = append(runs, cloneRun(run))
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].ID < runs[j].ID })
	return runs, nil
}

func cloneRun(run flow.Run) flow.Run {
	run.State.Data = append([]byte(nil), run.State.Data...)
	run.Steps = append([]flow.StepRecord(nil), run.Steps...)
	if run.Await != nil {
		await := *run.Await
		run.Await = &await
	}
	return run
}

type effectLedger struct {
	mu     sync.Mutex
	counts map[string]int
}

func newEffects() *effectLedger { return &effectLedger{counts: make(map[string]int)} }

func (l *effectLedger) apply(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.counts[key] == 0 {
		l.counts[key] = 1
	}
}

func (l *effectLedger) count(key string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.counts[key]
}

type barrier struct {
	mu      sync.Mutex
	want    int
	arrived int
	release chan struct{}
}

var finalBarrier = newBarrier(1)

func newBarrier(want int) *barrier { return &barrier{want: want, release: make(chan struct{})} }

func (b *barrier) arrive(_ int) {
	b.mu.Lock()
	b.arrived++
	if b.arrived == b.want {
		close(b.release)
	}
	release := b.release
	b.mu.Unlock()
	<-release
}
