package flow

import (
	"context"
	"time"

	"go-micro.dev/v6/ai"
)

// Schedule binds a flow to a recurring work item without introducing a
// scheduler service. It is a small harness contract: callers own the clock,
// Go Micro owns turning each tick into the same inspectable flow run used for
// broker events and direct Execute calls.
type Schedule struct {
	flow *Flow
	data string
}

// Scheduled returns a deterministic scheduled-run harness for this flow.
// Tests and event loops can call Tick directly; production processes can wire
// the same contract to time.Ticker through RunEvery. Each tick calls Execute, so
// checkpointed run history, parent/run metadata, cancellation, and inspection
// stay on the normal flow surfaces.
func Scheduled(f *Flow, data string) Schedule {
	return Schedule{flow: f, data: data}
}

// Tick starts one scheduled run immediately and returns when that run finishes.
func (s Schedule) Tick(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	info, _ := ai.RunInfoFrom(ctx)
	info.Dispatch = "schedule"
	if info.Trigger == "" {
		info.Trigger = s.flow.opts.TriggerTopic
	}
	if info.Trigger == "" {
		info.Trigger = "schedule"
	}
	return s.flow.Execute(ai.WithRunInfo(ctx, info), s.data)
}

// RunEvery drives scheduled runs from a ticker until ctx is canceled. It does
// not persist schedule definitions or host a scheduler; it only adapts a caller
// owned cadence to Tick.
func (s Schedule) RunEvery(ctx context.Context, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.Tick(ctx); err != nil {
				return err
			}
		}
	}
}
