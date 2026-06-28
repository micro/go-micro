package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/flow"
)

const (
	agentAskStep      = "ask"
	agentApprovalStep = "approval"
	agentInputStep    = "input-required"
)

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
		run.Steps[0].Result = ""
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

// Resume returns the response for a checkpointed agent run. Completed runs are
// returned from the checkpoint without calling the model or replaying tool
// calls; failed or in-progress runs continue from the saved input message.
func Resume(ctx context.Context, ag Agent, runID string) (*Response, error) {
	a, ok := ag.(*agentImpl)
	if !ok {
		return nil, fmt.Errorf("agent resume: unsupported agent implementation %T", ag)
	}
	return a.resume(ctx, runID)
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
	if run.Status == "paused" {
		if run.State.Stage == agentInputStep {
			return nil, fmt.Errorf("agent run %s is input-required; resume with ResumeInput", runID)
		}
		run.Status = "running"
		run.State.Stage = agentAskStep
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

// ResumeInput resumes a checkpointed agent run that paused via the built-in
// request_input tool. The supplied input is appended to the original request so
// the same run can continue with durable checkpoint and completed tool history.
func ResumeInput(ctx context.Context, ag Agent, runID, input string) (*Response, error) {
	a, ok := ag.(*agentImpl)
	if !ok {
		return nil, fmt.Errorf("agent resume input: unsupported agent implementation %T", ag)
	}
	return a.resumeInput(ctx, runID, input)
}

func (a *agentImpl) resumeInput(ctx context.Context, runID, input string) (*Response, error) {
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
	if run.Status != "paused" || run.State.Stage != agentInputStep {
		return nil, fmt.Errorf("agent run %s is not waiting for human input", runID)
	}
	var p inputPause
	if err := run.State.Scan(&p); err != nil {
		return nil, fmt.Errorf("agent run %s input state decode: %w", runID, err)
	}
	message := p.OriginalMessage
	if message == "" {
		message = string(run.State.Data)
	}
	message += "\n\nHuman input: " + input
	run.Status = "running"
	run.State.Stage = agentAskStep
	run.State.Data = []byte(message)
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.model == nil {
		a.setup()
	}
	return a.askLocked(ctx, run.ID, message, run.ParentID, &run)
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

func (a *agentImpl) checkpointToolWrap(next ai.ToolHandler) ai.ToolHandler {
	return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		if a.opts.Checkpoint == nil || a.currentRun == nil {
			return next(ctx, call)
		}
		name := toolCheckpointName(call)
		if rec, ok := findStep(a.currentRun.Steps, name); ok && rec.Status == "done" {
			return ai.ToolResult{ID: call.ID, Value: rec.Result, Content: rec.Result}
		}

		idx := upsertStep(&a.currentRun.Steps, flow.StepRecord{Name: name, Status: "in_progress"})
		_ = a.saveRun(ctx, *a.currentRun)
		res := next(ctx, call)
		a.currentRun.Steps[idx].Attempts++
		if res.Refused != "" {
			a.currentRun.Steps[idx].Status = "failed"
			a.currentRun.Steps[idx].Error = res.Content
			_ = a.saveRun(ctx, *a.currentRun)
			return res
		}
		a.currentRun.Steps[idx].Status = "done"
		a.currentRun.Steps[idx].Result = res.Content
		a.currentRun.Steps[idx].Error = ""
		_ = a.saveRun(ctx, *a.currentRun)
		return res
	}
}

func toolCheckpointName(call ai.ToolCall) string {
	b, _ := json.Marshal(call.Input)
	return "tool:" + call.Name + ":" + string(b)
}

func findStep(steps []flow.StepRecord, name string) (flow.StepRecord, bool) {
	for _, step := range steps {
		if step.Name == name {
			return step, true
		}
	}
	return flow.StepRecord{}, false
}

func upsertStep(steps *[]flow.StepRecord, rec flow.StepRecord) int {
	for i := range *steps {
		if (*steps)[i].Name == rec.Name {
			(*steps)[i].Status = rec.Status
			(*steps)[i].Error = rec.Error
			return i
		}
	}
	if len(*steps) == 0 || (*steps)[0].Name != agentAskStep {
		*steps = append([]flow.StepRecord{{Name: agentAskStep, Status: "in_progress"}}, (*steps)...)
	}
	*steps = append(*steps, rec)
	return len(*steps) - 1
}
