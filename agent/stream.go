package agent

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"

	"github.com/google/uuid"

	"go-micro.dev/v6/ai"
)

// StreamEventType identifies an event emitted by a tool-aware agent stream.
type StreamEventType string

const (
	// StreamEventToolStart is emitted immediately before a tool call runs.
	StreamEventToolStart StreamEventType = "tool_start"
	// StreamEventToolEnd is emitted after a tool call returns or is refused.
	StreamEventToolEnd StreamEventType = "tool_end"
	// StreamEventToken carries a chunk of the final answer.
	StreamEventToken StreamEventType = "token"
	// StreamEventDone carries the completed agent response.
	StreamEventDone StreamEventType = "done"
)

// StreamEvent is one event from StreamAsk.
type StreamEvent struct {
	Type     StreamEventType
	Token    string
	ToolCall ai.ToolCall
	Result   ai.ToolResult
	Response *Response
}

// AgentStream is a stream of tool execution events followed by final-answer chunks.
type AgentStream interface {
	Recv() (*StreamEvent, error)
	Close() error
}

// StreamAsk runs an agent Ask turn with tool start/end events and streams the final answer.
// It is additive for callers that hold the public Agent interface; concrete agents also
// expose the same method directly.
func StreamAsk(ctx context.Context, ag Agent, message string) (AgentStream, error) {
	streamer, ok := ag.(interface {
		StreamAsk(context.Context, string) (AgentStream, error)
	})
	if !ok {
		return nil, errors.New("agent: StreamAsk unsupported by implementation")
	}
	return streamer.StreamAsk(ctx, message)
}

// ResumeStreamAsk resumes a checkpointed agent run and emits the same event
// shape as StreamAsk. Completed runs are streamed from the persisted response;
// unfinished runs continue from their checkpoint and emit tool events for any
// work that still needs to run. Tool calls already recorded as done in the
// checkpoint are reused by the agent checkpoint wrapper and are not re-executed.
func ResumeStreamAsk(ctx context.Context, ag Agent, runID string) (AgentStream, error) {
	a, ok := ag.(*agentImpl)
	if !ok {
		return nil, errors.New("agent: ResumeStreamAsk unsupported by implementation")
	}
	return a.resumeStreamAsk(ctx, runID)
}

// StreamAsk runs tools like Ask, emits ToolStart/ToolEnd events as they execute,
// then emits chunks of the final answer followed by a Done event.
func (a *agentImpl) StreamAsk(ctx context.Context, message string) (AgentStream, error) {
	events := make(chan *StreamEvent, 16)
	done := make(chan struct{})
	s := &agentStream{events: events, done: done}

	go func() {
		defer close(events)
		defer close(done)
		resp, err := a.askWithStreamEvents(ctx, message, events)
		if err != nil {
			s.setErr(err)
			return
		}
		for _, tok := range splitStreamTokens(resp.Reply) {
			if !sendStreamEvent(ctx, events, &StreamEvent{Type: StreamEventToken, Token: tok}) {
				return
			}
		}
		_ = sendStreamEvent(ctx, events, &StreamEvent{Type: StreamEventDone, Response: resp})
	}()
	return s, nil
}

func (a *agentImpl) resumeStreamAsk(ctx context.Context, runID string) (AgentStream, error) {
	events := make(chan *StreamEvent, 16)
	done := make(chan struct{})
	s := &agentStream{events: events, done: done}

	go func() {
		defer close(events)
		defer close(done)
		resp, err := a.resumeWithStreamEvents(ctx, runID, events)
		if err != nil {
			s.setErr(err)
			return
		}
		for _, tok := range splitStreamTokens(resp.Reply) {
			if !sendStreamEvent(ctx, events, &StreamEvent{Type: StreamEventToken, Token: tok}) {
				return
			}
		}
		_ = sendStreamEvent(ctx, events, &StreamEvent{Type: StreamEventDone, Response: resp})
	}()
	return s, nil
}

func (a *agentImpl) askWithStreamEvents(ctx context.Context, message string, events chan<- *StreamEvent) (*Response, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.tools == nil {
		a.tools = ai.NewTools(a.opts.Registry, ai.ToolClient(a.opts.Client))
	}
	base := a.toolHandler()
	handler := func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		_ = sendStreamEvent(ctx, events, &StreamEvent{Type: StreamEventToolStart, ToolCall: call})
		result := base(ctx, call)
		_ = sendStreamEvent(ctx, events, &StreamEvent{Type: StreamEventToolEnd, ToolCall: call, Result: result})
		return result
	}
	a.setupWithToolHandler(handler)
	defer a.setupWithToolHandler(nil)
	return a.askLocked(ctx, uuid.New().String(), message, a.parentRunID, nil, true)
}

func (a *agentImpl) resumeWithStreamEvents(ctx context.Context, runID string, events chan<- *StreamEvent) (*Response, error) {
	if a.opts.Checkpoint == nil {
		return nil, errors.New("agent: ResumeStreamAsk requires a checkpoint")
	}
	run, ok, err := a.opts.Checkpoint.Load(ctx, runID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("agent: checkpointed run not found")
	}
	if run.Status == "done" {
		var resp Response
		if err := json.Unmarshal(run.State.Data, &resp); err != nil {
			return nil, err
		}
		return &resp, nil
	}
	if terminalAgentRunStatus(run.Status) {
		return nil, errors.New("agent: checkpointed run is terminal with status " + run.Status)
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.tools == nil {
		a.tools = ai.NewTools(a.opts.Registry, ai.ToolClient(a.opts.Client))
	}
	base := a.toolHandler()
	handler := func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		_ = sendStreamEvent(ctx, events, &StreamEvent{Type: StreamEventToolStart, ToolCall: call})
		result := base(ctx, call)
		_ = sendStreamEvent(ctx, events, &StreamEvent{Type: StreamEventToolEnd, ToolCall: call, Result: result})
		return result
	}
	a.setupWithToolHandler(handler)
	defer a.setupWithToolHandler(nil)
	if run.Status == "paused" {
		if run.State.Stage == agentInputStep {
			return nil, errors.New("agent: checkpointed run is input-required; resume with ResumeInput")
		}
		run.Status = "running"
		run.State.Stage = agentAskStep
	}
	return a.askLocked(ctx, run.ID, string(run.State.Data), run.ParentID, &run, false)
}

type agentStreamAdapter struct {
	stream AgentStream
}

type memoryRecordingStream struct {
	stream ai.Stream
	memory Memory

	mu     sync.Mutex
	chunks []string
	closed bool
}

func (s *memoryRecordingStream) Recv() (*ai.Response, error) {
	resp, err := s.stream.Recv()
	if resp != nil && resp.Reply != "" {
		s.mu.Lock()
		s.chunks = append(s.chunks, resp.Reply)
		s.mu.Unlock()
	}
	if errors.Is(err, io.EOF) {
		s.recordAssistant()
	}
	return resp, err
}

func (s *memoryRecordingStream) Close() error {
	s.recordAssistant()
	return s.stream.Close()
}

func (s *memoryRecordingStream) recordAssistant() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	if reply := strings.Join(s.chunks, ""); reply != "" {
		s.memory.Add("assistant", reply)
	}
}

func (s *agentStreamAdapter) Recv() (*ai.Response, error) {
	for {
		event, err := s.stream.Recv()
		if err != nil {
			return nil, err
		}
		if event == nil {
			continue
		}
		switch event.Type {
		case StreamEventToken:
			if event.Token == "" {
				continue
			}
			return &ai.Response{Reply: event.Token}, nil
		case StreamEventDone:
			return nil, io.EOF
		}
	}
}

func (s *agentStreamAdapter) Close() error {
	return s.stream.Close()
}

func (a *agentImpl) streamAskAI(ctx context.Context, message string) (ai.Stream, error) {
	stream, err := a.StreamAsk(ctx, message)
	if err != nil {
		return nil, err
	}
	return &agentStreamAdapter{stream: stream}, nil
}

type agentStream struct {
	events <-chan *StreamEvent
	done   <-chan struct{}
	mu     sync.Mutex
	err    error
}

func (s *agentStream) Recv() (*StreamEvent, error) {
	ev, ok := <-s.events
	if ok {
		return ev, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return nil, s.err
	}
	return nil, io.EOF
}

func (s *agentStream) Close() error {
	<-s.done
	return nil
}

func (s *agentStream) setErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.err = err
}

func sendStreamEvent(ctx context.Context, events chan<- *StreamEvent, ev *StreamEvent) bool {
	select {
	case events <- ev:
		return true
	case <-ctx.Done():
		return false
	}
}

func splitStreamTokens(reply string) []string {
	if reply == "" {
		return nil
	}
	parts := strings.Fields(reply)
	if len(parts) == 0 {
		return []string{reply}
	}
	out := make([]string, 0, len(parts))
	for i, part := range parts {
		if i > 0 {
			part = " " + part
		}
		out = append(out, part)
	}
	return out
}
