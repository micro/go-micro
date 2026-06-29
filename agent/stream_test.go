package agent

import (
	"context"
	"errors"
	"io"
	"testing"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/flow"
	"go-micro.dev/v6/store"
)

func TestStreamAskEmitsToolEventsAndFinalTokens(t *testing.T) {
	calls := 0
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		if opts.ToolHandler == nil {
			t.Fatal("StreamAsk must configure a tool handler")
		}
		calls++
		result := opts.ToolHandler(ctx, ai.ToolCall{ID: "call-1", Name: "echo", Input: map[string]any{"text": "hello"}})
		return &ai.Response{
			Reply:     "planning",
			Answer:    "final answer",
			ToolCalls: []ai.ToolCall{{ID: "call-1", Name: "echo", Input: map[string]any{"text": "hello"}, Result: result.Content}},
		}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("streamer"), WithTool("echo", "echo text", nil, func(ctx context.Context, input map[string]any) (string, error) {
		return input["text"].(string), nil
	}))
	stream, err := a.StreamAsk(context.Background(), "say hello")
	if err != nil {
		t.Fatalf("StreamAsk: %v", err)
	}

	var types []StreamEventType
	var tokens string
	var done *Response
	for {
		event, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
		types = append(types, event.Type)
		if event.Type == StreamEventToken {
			tokens += event.Token
		}
		if event.Type == StreamEventDone {
			done = event.Response
		}
	}

	want := []StreamEventType{StreamEventToolStart, StreamEventToolEnd, StreamEventToken, StreamEventToken, StreamEventToken, StreamEventDone}
	if len(types) != len(want) {
		t.Fatalf("event types = %v, want %v", types, want)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("event types = %v, want %v", types, want)
		}
	}
	if tokens != "planning final answer" {
		t.Fatalf("tokens = %q", tokens)
	}
	if done == nil || done.Reply != "planning\n\nfinal answer" {
		t.Fatalf("done response = %#v", done)
	}
	if calls != 1 {
		t.Fatalf("Generate calls = %d, want 1", calls)
	}
}

func TestStreamAskHelperRejectsUnsupportedAgent(t *testing.T) {
	_, err := StreamAsk(context.Background(), unsupportedAgent{}, "hello")
	if err == nil {
		t.Fatal("StreamAsk helper should reject unsupported implementations")
	}
}

func TestResumeStreamAskDoesNotReplayCompletedTool(t *testing.T) {
	ctx := context.Background()
	cp := flow.StoreCheckpoint(store.NewStore(), "stream-resume-agent")
	toolRuns := 0
	first := true
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		if opts.ToolHandler != nil {
			res := opts.ToolHandler(ctx, ai.ToolCall{ID: "call-1", Name: "charge", Input: map[string]any{"order": "42"}})
			if res.Content != "charged" {
				t.Fatalf("tool result = %q, want charged", res.Content)
			}
		}
		if first {
			first = false
			return nil, errors.New("stream disconnected after tool")
		}
		return &ai.Response{Reply: "finished from streamed checkpoint"}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("stream-resume-agent"), WithCheckpoint(cp),
		WithTool("charge", "charge once", nil, func(context.Context, map[string]any) (string, error) {
			toolRuns++
			return "charged", nil
		}))
	stream, err := a.StreamAsk(ctx, "charge order 42")
	if err != nil {
		t.Fatalf("StreamAsk: %v", err)
	}
	for {
		_, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			break
		}
	}
	if toolRuns != 1 {
		t.Fatalf("tool executions after failed StreamAsk = %d, want 1", toolRuns)
	}
	runs, err := Pending(ctx, a)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("Pending returned %d runs, want 1", len(runs))
	}

	resumed, err := ResumeStreamAsk(ctx, a, runs[0].ID)
	if err != nil {
		t.Fatalf("ResumeStreamAsk: %v", err)
	}
	var toolEvents int
	var done *Response
	for {
		event, err := resumed.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("resumed Recv: %v", err)
		}
		if event.Type == StreamEventToolStart || event.Type == StreamEventToolEnd {
			toolEvents++
		}
		if event.Type == StreamEventDone {
			done = event.Response
		}
	}
	if toolRuns != 1 {
		t.Fatalf("tool executions after ResumeStreamAsk = %d, want completed tool was not replayed", toolRuns)
	}
	if toolEvents != 2 {
		t.Fatalf("resumed tool events = %d, want start/end for replayed checkpoint result", toolEvents)
	}
	if done == nil || done.Reply != "finished from streamed checkpoint" || done.RunID != runs[0].ID {
		t.Fatalf("done response = %#v", done)
	}
}

type unsupportedAgent struct{}

func (unsupportedAgent) Name() string                                      { return "unsupported" }
func (unsupportedAgent) Init(...Option)                                    {}
func (unsupportedAgent) Options() Options                                  { return Options{} }
func (unsupportedAgent) Ask(context.Context, string) (*Response, error)    { return nil, nil }
func (unsupportedAgent) Stream(context.Context, string) (ai.Stream, error) { return nil, nil }
func (unsupportedAgent) Run() error                                        { return nil }
func (unsupportedAgent) Stop() error                                       { return nil }
func (unsupportedAgent) String() string                                    { return "unsupported" }
