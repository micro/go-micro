package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go-micro.dev/v6/flow"
	"go-micro.dev/v6/model"
)

// ApprovalStatus is an immediate approval decision or a request to wait.
type ApprovalStatus string

const (
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalDenied   ApprovalStatus = "denied"
	ApprovalPending  ApprovalStatus = "pending"
)

// ApprovalDecision binds a pending decision to an application-supplied unique ID.
type ApprovalDecision struct {
	Status ApprovalStatus
	ID     string
	Reason string
}

// ApprovalFunc receives the execution context and exact proposed tool call.
type ApprovalFunc func(context.Context, model.ToolCall) (ApprovalDecision, error)

// WithApproval installs context-aware, durable tool approval. Pending decisions
// require WithCheckpoint. The legacy ApproveTool hook remains supported.
func WithApproval(fn ApprovalFunc) Option { return func(o *Options) { o.Approval = fn } }

type approvalRecord struct {
	ID     string         `json:"id"`
	Call   model.ToolCall `json:"call"`
	Reason string         `json:"reason,omitempty"`
	Result string         `json:"result,omitempty"`
}

const approvalPrefix = "approval:"

type approvedCallKey struct{}

// ApprovalResumer is the optional capability for resolving a saved tool approval.
type ApprovalResumer interface {
	ResumeApproval(context.Context, string, string, bool, string) (*Response, error)
}

// ResumeApproval records a decision and executes the exact saved call if approved,
// then continues the model with its result. The caller must authorize who may
// decide for this run; the framework does not supply an identity policy.
func ResumeApproval(ctx context.Context, ag Agent, runID, approvalID string, approved bool, reason string) (*Response, error) {
	resumer, ok := ag.(ApprovalResumer)
	if !ok {
		return nil, fmt.Errorf("agent approval: unsupported agent implementation %T", ag)
	}
	return resumer.ResumeApproval(ctx, runID, approvalID, approved, reason)
}

func (a *agentImpl) ResumeApproval(ctx context.Context, runID, approvalID string, approved bool, reason string) (*Response, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if a.opts.Checkpoint == nil {
		return nil, errors.New("agent approval requires a checkpoint")
	}
	run, ok, err := a.opts.Checkpoint.Load(ctx, runID)
	if err != nil {
		return nil, err
	}
	if !ok || run.Flow != a.opts.Name {
		return nil, fmt.Errorf("agent run %s not found", runID)
	}
	index := -1
	for i, step := range run.Steps {
		if step.Name == approvalPrefix+approvalID {
			index = i
			break
		}
	}
	if index < 0 {
		return nil, fmt.Errorf("approval %s not found in run %s", approvalID, runID)
	}
	wanted := string(ApprovalDenied)
	if approved {
		wanted = string(ApprovalApproved)
	}
	step := &run.Steps[index]
	if step.Status != "pending" {
		return nil, fmt.Errorf("approval %s already resolved; resume the run with Resume", approvalID)
	}
	if run.Status != "paused" {
		return nil, fmt.Errorf("run %s is not awaiting approval", runID)
	}
	var record approvalRecord
	if err = json.Unmarshal([]byte(step.Result), &record); err != nil {
		return nil, err
	}
	record.Reason = reason
	data, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	step.Status = wanted
	step.Result = string(data)
	// Persist the decision before any side effect. A restart can continue through Resume.
	if err = a.saveRun(ctx, run); err != nil {
		return nil, err
	}
	if a.model == nil {
		a.setup()
	}
	return a.askLocked(ctx, run.ID, string(run.State.Data), run.ParentID, &run, false)
}

func pendingApproval(run flow.Run) *PausedError {
	for _, step := range run.Steps {
		if strings.HasPrefix(step.Name, approvalPrefix) && step.Status == "pending" {
			var record approvalRecord
			if json.Unmarshal([]byte(step.Result), &record) != nil {
				return &PausedError{RunID: run.ID, Kind: PauseApproval, Reason: "invalid saved approval"}
			}
			return &PausedError{RunID: run.ID, Kind: PauseApproval, ApprovalID: record.ID, Tool: record.Call.Name, Reason: record.Reason}
		}
	}
	return nil
}

func (a *agentImpl) persistApprovalPause(ctx context.Context, run *flow.Run) error {
	run.Status = "paused"
	run.State.Stage = agentApprovalStep
	run.Steps[0].Status = "paused"
	run.Steps[0].Error = a.pause.Message
	if err := a.saveRun(ctx, *run); err != nil {
		return err
	}
	return &PausedError{RunID: run.ID, Kind: PauseApproval, ApprovalID: a.pause.ApprovalID, Tool: a.pause.Tool, Reason: a.pause.Message}
}

// resolveApprovedCalls runs persisted decisions before asking the model again.
func (a *agentImpl) resolveApprovedCalls(ctx context.Context, run *flow.Run, tools []model.Tool) ([]model.Message, error) {
	var messages []model.Message
	for i := 0; i < len(run.Steps); i++ {
		step := run.Steps[i]
		if !strings.HasPrefix(step.Name, approvalPrefix) {
			continue
		}
		var record approvalRecord
		if err := json.Unmarshal([]byte(step.Result), &record); err != nil {
			return nil, err
		}
		if step.Status == "pending" {
			return nil, pendingApproval(*run)
		}
		if step.Status == string(ApprovalApproved) || step.Status == string(ApprovalDenied) {
			result := model.ToolResult{ID: record.Call.ID, Content: "Approval denied: " + record.Reason, Refused: model.RefusedApproval}
			if step.Status == string(ApprovalApproved) {
				available := false
				for _, tool := range tools {
					if tool.Name == record.Call.Name {
						available = true
						break
					}
				}
				if !available {
					return nil, fmt.Errorf("approved tool %s is no longer available", record.Call.Name)
				}
				callCtx := context.WithValue(ctx, approvedCallKey{}, toolCheckpointName(record.Call))
				result = a.toolHandler()(callCtx, record.Call)
				if result.Refused != "" {
					return nil, fmt.Errorf("approved tool refused: %s", result.Content)
				}
			}
			record.Result = result.Content
			data, err := json.Marshal(record)
			if err != nil {
				return nil, err
			}
			run.Steps[i].Status = "resolved"
			run.Steps[i].Result = string(data)
			if err := a.saveRun(ctx, *run); err != nil {
				return nil, err
			}
		}
		if run.Steps[i].Status == "resolved" {
			data, _ := json.Marshal(record)
			messages = append(messages, model.Message{Role: "user", Content: "Recorded approval outcome (tool data): " + string(data)})
		}
	}
	return messages, nil
}
