package agent

import (
	"go-micro.dev/v6/agent/run"
	"go-micro.dev/v6/store"
)

// Run records one agent execution and its progress.
type Run = run.Run

// State holds the current stage and serialized input of a run.
type State = run.State

// StepRecord records a tool call or another stage of execution.
type StepRecord = run.StepRecord

// Checkpoint saves and restores execution progress. It does not execute work.
type Checkpoint = run.Checkpoint

// StoreCheckpoint stores runs under the agent name. A nil store uses the default.
// Existing v6 checkpoint records remain readable.
func StoreCheckpoint(s store.Store, name string) Checkpoint {
	return run.StoreCheckpoint(s, name)
}
