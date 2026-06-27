package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-micro.dev/v6/flow"
)

const agentAskStep = "ask"

func (a *agentImpl) newCheckpointRun(runID, message, parentRunID string, existing *flow.Run) flow.Run {
	now := time.Now()
	run := flow.Run{
		ID:       runID,
		ParentID: parentRunID,
		Flow:     a.opts.Name,
		State:    flow.State{Stage: agentAskStep, Data: []byte(message)},
		Steps:    []flow.StepRecord{{Name: agentAskStep, Status: "in_progress"}},
		Status:   "running",
		Started:  now,
		Updated:  now,
	}
	if existing != nil {
		run = *existing
		run.Status = "running"
		run.State.Stage = agentAskStep
		if len(run.Steps) == 0 {
			run.Steps = []flow.StepRecord{{Name: agentAskStep}}
		}
		run.Steps[0].Status = "in_progress"
		run.Steps[0].Error = ""
	}
	return run
}

func (a *agentImpl) saveRun(ctx context.Context, run flow.Run) error {
	if a.opts.Checkpoint == nil {
		return nil
	}
	if err := a.opts.Checkpoint.Save(ctx, run); err != nil {
		return fmt.Errorf("agent %s checkpoint save: %w", a.opts.Name, err)
	}
	return nil
}

func (a *agentImpl) resume(ctx context.Context, runID string) (*Response, error) {
	if a.opts.Checkpoint == nil {
		return nil, fmt.Errorf("agent %s has no checkpoint configured", a.opts.Name)
	}
	run, ok, err := a.opts.Checkpoint.Load(ctx, runID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("agent run %s not found", runID)
	}
	if run.Status == "done" {
		var resp Response
		if err := json.Unmarshal(run.State.Data, &resp); err != nil {
			return nil, fmt.Errorf("agent run %s response decode: %w", runID, err)
		}
		return &resp, nil
	}
	message := string(run.State.Data)
	parentID := run.ParentID
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.model == nil {
		a.setup()
	}
	return a.askLocked(ctx, run.ID, message, parentID, &run)
}

func (a *agentImpl) pending(ctx context.Context) ([]flow.Run, error) {
	if a.opts.Checkpoint == nil {
		return nil, nil
	}
	runs, err := a.opts.Checkpoint.List(ctx)
	if err != nil {
		return nil, err
	}
	out := runs[:0]
	for _, run := range runs {
		if run.Flow == a.opts.Name && run.Status != "done" {
			out = append(out, run)
		}
	}
	return out, nil
}
