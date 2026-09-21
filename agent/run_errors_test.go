package agent

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go-micro.dev/v6/flow"
	"go-micro.dev/v6/model"
	"go-micro.dev/v6/store"
)

func TestTypedRunErrors(t *testing.T) {
	ctx := context.Background()
	cp := flow.StoreCheckpoint(store.NewMemoryStore(), "typed-errors")
	a := newTestAgent(Name("typed-errors"), WithCheckpoint(cp))
	for _, status := range []string{"timeout", "canceled", "rate_limited", "expired"} {
		if err := cp.Save(ctx, flow.Run{ID: status, Status: status}); err != nil {
			t.Fatal(err)
		}
		_, err := Resume(ctx, a, status)
		var terminal *TerminalRunError
		if !errors.Is(err, ErrRunTerminal) || !errors.As(err, &terminal) || terminal.RunID != status || terminal.Status != status {
			t.Fatalf("unexpected terminal error: %v", err)
		}
	}
	if err := cp.Save(ctx, flow.Run{ID: "input", Status: "paused", State: flow.State{Stage: agentInputStep}}); err != nil {
		t.Fatal(err)
	}
	_, err := Resume(ctx, a, "input")
	var input *AwaitingInputError
	if !errors.Is(err, ErrRunAwaitingInput) || !errors.As(err, &input) || input.RunID != "input" {
		t.Fatalf("unexpected input error: %v", err)
	}
}

func TestTypedMultilinePause(t *testing.T) {
	ctx := context.Background()
	cp := flow.StoreCheckpoint(store.NewMemoryStore(), "typed-pause")
	reason := "First question?\nSecond question?"
	fakeGen = func(ctx context.Context, opts model.Options, req *model.Request) (*model.Response, error) {
		opts.ToolHandler(ctx, model.ToolCall{ID: "ask-human", Name: toolHumanInput, Input: map[string]any{"prompt": reason}})
		return &model.Response{Reply: reason}, nil
	}
	defer func() { fakeGen = nil }()
	a := newTestAgent(Name("typed-pause"), WithCheckpoint(cp))
	_, err := a.Ask(ctx, "help")
	wrapped := fmt.Errorf("handler: %w", err)
	var pause *PausedError
	if !errors.As(wrapped, &pause) || !errors.Is(wrapped, ErrRunPaused) || !errors.Is(wrapped, ErrRunAwaitingInput) {
		t.Fatalf("unexpected pause: %v", err)
	}
	if pause.Kind != PauseInput || pause.Tool != toolHumanInput || pause.Reason != reason || pause.RunID == "" {
		t.Fatalf("pause details: %+v", pause)
	}
}
