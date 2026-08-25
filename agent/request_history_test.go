package agent

import (
	"context"
	"testing"

	"go-micro.dev/v6/ai"
)

// countPromptOccurrences counts how many times message reaches the provider in
// one request, across both channels providers replay: the Messages history and
// the Prompt field. Providers build their payload as Messages followed by
// Prompt, so the correct count for the turn being answered is exactly one.
func countPromptOccurrences(req *ai.Request, message string) int {
	seen := 0
	if req.Prompt == message {
		seen++
	}
	for _, msg := range req.Messages {
		if s, ok := msg.Content.(string); ok && msg.Role == "user" && s == message {
			seen++
		}
	}
	return seen
}

// The message being answered must reach the provider exactly once, even with
// prior conversation history in memory. Regression test for the double-send
// where memory recorded the turn before the request was built and the request
// then carried it both as the trailing history entry and as Prompt.
func TestCurrentMessageReachesProviderOnce(t *testing.T) {
	var last *ai.Request
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		last = req
		return &ai.Response{Reply: "ok"}, nil
	}
	defer func() { fakeGen = nil }()

	mem := NewInMemory(8)
	mem.Add("user", "first question")
	mem.Add("assistant", "first answer")

	a := newTestAgent(Name("dedupe-ask"), WithMemory(mem))
	if _, err := a.Ask(context.Background(), "the new question"); err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if last == nil {
		t.Fatal("provider never called")
	}
	if got := countPromptOccurrences(last, "the new question"); got != 1 {
		t.Errorf("the message reached the provider %d times, want 1 (Prompt=%q, Messages=%+v)",
			got, last.Prompt, last.Messages)
	}
	// The history itself must still be there.
	if len(last.Messages) < 2 {
		t.Errorf("history = %+v, want the prior turns preserved", last.Messages)
	}
}

// Same contract on the streaming path.
func TestCurrentMessageReachesProviderOnceStreaming(t *testing.T) {
	var last *ai.Request
	fakeStream = func(ctx context.Context, opts ai.Options, req *ai.Request) (ai.Stream, error) {
		last = req
		return &sliceStream{chunks: []string{"ok"}}, nil
	}
	defer func() { fakeStream = nil }()

	mem := NewInMemory(8)
	mem.Add("user", "first question")
	mem.Add("assistant", "first answer")

	a := newTestAgent(Name("dedupe-stream"), WithMemory(mem))
	stream, err := a.Stream(context.Background(), "the new question")
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	defer stream.Close()
	if last == nil {
		t.Fatal("provider never called")
	}
	if got := countPromptOccurrences(last, "the new question"); got != 1 {
		t.Errorf("the message reached the provider %d times, want 1 (Prompt=%q, Messages=%+v)",
			got, last.Prompt, last.Messages)
	}
}

func TestRequestHistoryTrimsOnlyTrailingCurrentTurn(t *testing.T) {
	history := []ai.Message{
		{Role: "user", Content: "a"},
		{Role: "assistant", Content: "b"},
	}
	withCurrent := append(append([]ai.Message(nil), history...), ai.Message{Role: "user", Content: "c"})

	if got := requestHistory(withCurrent, "c"); len(got) != 2 {
		t.Errorf("trailing current turn not trimmed: %+v", got)
	}
	if got := requestHistory(history, "c"); len(got) != 2 {
		t.Errorf("history without the current turn must be untouched: %+v", got)
	}
	// A user coincidentally repeating an EARLIER message must not lose history:
	// only a trailing entry equal to the current message is the double-send.
	if got := requestHistory(withCurrent, "a"); len(got) != 3 {
		t.Errorf("non-trailing repeat wrongly trimmed: %+v", got)
	}
	if got := requestHistory(nil, "c"); len(got) != 0 {
		t.Errorf("nil history: %+v", got)
	}
	if got := requestHistory(withCurrent, ""); len(got) != 3 {
		t.Errorf("empty message must not trim: %+v", got)
	}
}
