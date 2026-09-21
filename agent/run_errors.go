package agent

import (
	"errors"
	"fmt"
)

// Run-state sentinels support errors.Is; the concrete errors expose run details.
var (
	ErrRunPaused        = errors.New("agent run paused")
	ErrRunAwaitingInput = errors.New("agent run awaiting input")
	ErrRunTerminal      = errors.New("agent run terminal")
)

// PauseKind identifies the decision needed to continue a run.
type PauseKind string

const (
	PauseApproval PauseKind = "approval"
	PauseInput    PauseKind = "input"
)

// PausedError reports a persisted pause, including multiline human-facing text.
type PausedError struct {
	RunID  string
	Kind   PauseKind
	Tool   string
	Reason string
}

func (e *PausedError) Error() string {
	return fmt.Sprintf("agent run %s paused for approval: %s", e.RunID, e.Reason)
}
func (e *PausedError) Is(target error) bool {
	return target == ErrRunPaused || (e.Kind == PauseInput && target == ErrRunAwaitingInput)
}

// AwaitingInputError reports a run that must be resumed with ResumeInput.
type AwaitingInputError struct {
	RunID   string
	message string
}

func (e *AwaitingInputError) Error() string {
	if e.message != "" {
		return e.message
	}
	return fmt.Sprintf("agent run %s is input-required; resume with ResumeInput", e.RunID)
}
func (e *AwaitingInputError) Unwrap() error { return ErrRunAwaitingInput }

// TerminalRunError reports a run that cannot be resumed in its current state.
type TerminalRunError struct {
	RunID   string
	Status  string
	message string
}

func (e *TerminalRunError) Error() string {
	if e.message != "" {
		return e.message
	}
	return fmt.Sprintf("agent run %s is terminal with status %q", e.RunID, e.Status)
}
func (e *TerminalRunError) Unwrap() error { return ErrRunTerminal }
