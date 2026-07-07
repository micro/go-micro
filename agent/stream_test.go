package agent

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

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

func TestStreamAskCloseCancelsInFlightModelCall(t *testing.T) {
	started := make(chan struct{})
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("stream-cancel"))
	stream, err := a.StreamAsk(context.Background(), "cancel me")
	if err != nil {
		t.Fatalf("StreamAsk: %v", err)
	}

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("model call did not start")
	}

	closed := make(chan error, 1)
	go func() { closed <- stream.Close() }()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatalf("Close: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close did not cancel the in-flight stream")
	}
}

func TestStreamAskHelperRejectsUnsupportedAgent(t *testing.T) {
	_, err := StreamAsk(context.Background(), unsupportedAgent{}, "hello")
	if err == nil {
		t.Fatal("StreamAsk helper should reject unsupported implementations")
	}
}

func TestAgentStreamUsesProviderStreamingAndRecordsAssistantMemory(t *testing.T) {
	var sawRequest bool
	fakeStream = func(ctx context.Context, opts ai.Options, req *ai.Request) (ai.Stream, error) {
		sawRequest = true
		if req.Prompt != "stream the answer" {
			t.Fatalf("Prompt = %q, want stream the answer", req.Prompt)
		}
		if len(req.Messages) != 1 || req.Messages[0].Role != "user" || req.Messages[0].Content != "stream the answer" {
			t.Fatalf("Messages = %#v, want current user turn in memory", req.Messages)
		}
		return &sliceStream{chunks: []string{"hel", "lo"}}, nil
	}
	defer func() { fakeStream = nil }()

	a := newTestAgent(Name("provider-stream"))
	stream, err := a.Stream(context.Background(), "stream the answer")
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	var reply string
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
		reply += chunk.Reply
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !sawRequest {
		t.Fatal("provider Stream was not called")
	}
	if reply != "hello" {
		t.Fatalf("reply = %q, want hello", reply)
	}
	got := a.mem.Messages()
	if len(got) != 2 || got[0].Role != "user" || got[0].Content != "stream the answer" || got[1].Role != "assistant" || got[1].Content != "hello" {
		t.Fatalf("memory = %#v, want user turn and streamed assistant reply", got)
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
