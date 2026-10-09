package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"go-micro.dev/v6/agent"
	pb "go-micro.dev/v6/agent/proto"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/model"
)

func (s *session) interactAgent(ctx context.Context, name, message string) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	stream, err := pb.NewAgentInteractionService(name, s.cl).Chat(ctx, client.WithRetries(0))
	if err != nil {
		return err
	}
	defer stream.Close()
	if err := stream.Send(&pb.InteractionRequest{Message: message}); err != nil {
		return err
	}
	if s.display != nil {
		defer s.display.flush()
	}
	incoming := make(chan *pb.InteractionEvent, 1)
	go func() {
		for {
			event, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					err = errors.New("agent disconnected before finishing; completed actions may still have taken effect")
				}
				cancel(err)
				return
			}
			select {
			case incoming <- event:
			case <-ctx.Done():
				return
			}
			if event.Type == string(agent.StreamEventDone) {
				return
			}
		}
	}()
	for {
		var event *pb.InteractionEvent
		select {
		case event = <-incoming:
		case <-ctx.Done():
			return context.Cause(ctx)
		}
		if event.Type == string(agent.StreamEventDone) {
			if s.display == nil {
				fmt.Fprintln(s.writer())
			}
			return nil
		}
		call := model.ToolCall{}
		if event.ToolCall != nil {
			call.ID, call.Name = event.ToolCall.Id, event.ToolCall.Name
			if err := json.Unmarshal([]byte(event.ToolCall.Input), &call.Input); err != nil {
				return err
			}
		}
		if event.Type == string(agent.StreamEventApproval) {
			if s.display != nil {
				s.display.flush()
			}
			decision, err := s.promptApproval(ctx, call)
			if err != nil {
				if ctx.Err() != nil {
					return context.Cause(ctx)
				}
				return err
			}
			if err := stream.Send(&pb.InteractionRequest{ApprovalId: event.ApprovalId, Approved: decision.Status == agent.ApprovalApproved}); err != nil {
				return err
			}
			continue
		}
		s.showEvent(&agent.StreamEvent{Type: agent.StreamEventType(event.Type), Token: event.Text, ToolCall: call, Result: model.ToolResult{Content: event.Result}})
	}
}
