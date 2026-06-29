package agent

import (
	"context"
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
	return a.askLocked(ctx, uuid.New().String(), message, a.parentRunID, nil, true)
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
