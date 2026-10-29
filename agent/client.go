package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	pb "go-micro.dev/v6/agent/proto"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/model"
)

// Client calls a registered agent. Use WithSession to select a conversation.
// Model configuration and tool permissions remain with the host.
type Client interface {
	Ask(context.Context, string) (*Response, error)
	StreamAsk(context.Context, string) (ClientStream, error)
}

// ClientStream receives agent events and answers approval requests.
// Context is cancelled when the connection fails, including while waiting for
// approval. Close cancels unfinished work. Always close the stream when done.
type ClientStream interface {
	AgentStream
	Context() context.Context
	Approve(id string, approved bool) error
}

type agentClient struct {
	name   string
	client client.Client
}

// NewClient returns a client for name using c for discovery and RPC. A nil c uses the
// default RPC client. Calls are not retried: tools may have already taken effect.
func NewClient(name string, c client.Client) Client {
	if c == nil {
		c = client.DefaultClient
	}
	return &agentClient{name: name, client: c}
}

func (c *agentClient) Ask(ctx context.Context, message string) (*Response, error) {
	response, err := pb.NewAgentService(c.name, c.client).Chat(ctx, &pb.ChatRequest{Message: message}, client.WithRetries(0))
	if err != nil {
		return nil, err
	}
	return decodeResponse(response)
}

// StreamAsk requires the host's AgentInteraction endpoint. It never falls back
// to another endpoint after dispatching a request.
func (c *agentClient) StreamAsk(ctx context.Context, message string) (ClientStream, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	stream, err := pb.NewAgentInteractionService(c.name, c.client).Chat(ctx, client.WithRetries(0))
	if err != nil {
		cancel(err)
		return nil, err
	}
	if err := stream.Send(&pb.InteractionRequest{Message: message}); err != nil {
		cancel(err)
		_ = stream.Close()
		return nil, err
	}
	s := &remoteStream{ctx: ctx, cancel: cancel, stream: stream, events: make(chan *StreamEvent, 1)}
	stop := context.AfterFunc(ctx, func() { _ = s.closeTransport() })
	go func() { defer stop(); s.receive() }()
	return s, nil
}

type remoteStream struct {
	ctx      context.Context
	cancel   context.CancelCauseFunc
	stream   pb.AgentInteraction_ChatService
	events   chan *StreamEvent
	once     sync.Once
	closeErr error
}

func (s *remoteStream) Context() context.Context { return s.ctx }

func (s *remoteStream) Approve(id string, approved bool) error {
	if err := context.Cause(s.ctx); err != nil {
		return err
	}
	if id == "" {
		return errors.New("agent: approval ID is required")
	}
	return s.stream.Send(&pb.InteractionRequest{ApprovalId: id, Approved: approved})
}

func (s *remoteStream) closeTransport() error {
	s.once.Do(func() { s.closeErr = s.stream.Close() })
	return s.closeErr
}

func (s *remoteStream) Close() error {
	s.cancel(context.Canceled)
	return s.closeTransport()
}

func (s *remoteStream) Recv() (*StreamEvent, error) {
	select {
	case event, ok := <-s.events:
		if ok {
			return event, nil
		}
		if err := context.Cause(s.ctx); err != nil {
			return nil, err
		}
		return nil, io.EOF
	case <-s.ctx.Done():
		return nil, context.Cause(s.ctx)
	}
}

func (s *remoteStream) receive() {
	defer close(s.events)
	defer s.closeTransport()
	for {
		event, err := s.stream.Recv()
		if errors.Is(err, io.EOF) {
			err = errors.New("agent disconnected before finishing; completed actions may still have taken effect")
		}
		if err != nil {
			s.cancel(err)
			return
		}
		if event == nil {
			s.cancel(errors.New("agent: empty stream event"))
			return
		}
		call, err := decodeToolCall(event.ToolCall)
		if err != nil {
			s.cancel(err)
			return
		}
		out := &StreamEvent{Type: StreamEventType(event.Type), Token: event.Text, ApprovalID: event.ApprovalId, ToolCall: call, Result: model.ToolResult{Content: event.Result}}
		if event.Response != nil {
			out.Response, err = decodeResponse(event.Response)
			if err != nil {
				s.cancel(err)
				return
			}
		}
		select {
		case s.events <- out:
		case <-s.ctx.Done():
			return
		}
		if out.Type == StreamEventDone {
			return
		}
	}
}

func decodeToolCall(call *pb.ToolCall) (model.ToolCall, error) {
	if call == nil {
		return model.ToolCall{}, nil
	}
	out := model.ToolCall{ID: call.Id, Name: call.Name, Result: call.Result}
	if call.Input != "" {
		if err := json.Unmarshal([]byte(call.Input), &out.Input); err != nil {
			return out, fmt.Errorf("agent: decode tool %q: %w", call.Name, err)
		}
	}
	return out, nil
}

func decodeResponse(response *pb.ChatResponse) (*Response, error) {
	if response == nil {
		return nil, errors.New("agent: empty response")
	}
	out := &Response{Reply: response.Reply, Agent: response.Agent, RunID: response.RunId, ParentID: response.ParentId}
	for _, call := range response.ToolCalls {
		tool, err := decodeToolCall(call)
		if err != nil {
			return nil, err
		}
		out.ToolCalls = append(out.ToolCalls, tool)
	}
	return out, nil
}
