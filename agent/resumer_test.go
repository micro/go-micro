package agent

import (
	"context"
	"errors"
	"testing"
)

type resumeFake struct {
	Agent
	ctx      context.Context
	runID    string
	input    string
	err      error
	response *Response
}

func (f *resumeFake) Resume(ctx context.Context, runID string) (*Response, error) {
	f.ctx, f.runID = ctx, runID
	return f.response, f.err
}
func (f *resumeFake) ResumeInput(ctx context.Context, runID, input string) (*Response, error) {
	f.ctx, f.runID, f.input = ctx, runID, input
	return f.response, f.err
}
func (f *resumeFake) ResumeStreamAsk(ctx context.Context, runID string) (AgentStream, error) {
	f.ctx, f.runID = ctx, runID
	return nil, f.err
}
func TestResumeCapabilities(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	expected := errors.New("resume sentinel")
	fake := &resumeFake{err: expected, response: &Response{RunID: "saved"}}
	response, err := Resume(ctx, fake, "saved")
	if response != fake.response || err != expected || fake.ctx != ctx || fake.runID != "saved" {
		t.Fatalf("Resume did not forward to the implementation: %v", err)
	}
	response, err = ResumeInput(ctx, fake, "input-run", "human input")
	if response != fake.response || err != expected || fake.ctx != ctx || fake.runID != "input-run" || fake.input != "human input" {
		t.Fatalf("ResumeInput did not forward to the implementation: %v", err)
	}
	_, err = ResumeStreamAsk(ctx, fake, "stream-run")
	if err != expected || fake.ctx != ctx || fake.runID != "stream-run" {
		t.Fatalf("ResumeStreamAsk did not forward to the implementation: %v", err)
	}
	// Existing implementations need not implement any optional capability.
	unsupported := struct{ Agent }{}
	if _, err := Resume(ctx, unsupported, "saved"); err == nil {
		t.Fatal("expected unsupported error")
	}
	if _, err := ResumeInput(ctx, unsupported, "saved", "input"); err == nil {
		t.Fatal("expected unsupported error")
	}
	if _, err := ResumeStreamAsk(ctx, unsupported, "saved"); err == nil {
		t.Fatal("expected unsupported error")
	}
}

var _ Resumer = (*agentImpl)(nil)
var _ InputResumer = (*agentImpl)(nil)
var _ StreamResumer = (*agentImpl)(nil)
