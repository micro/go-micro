package chat

import (
	"context"
	"errors"
	"fmt"
	"io"

	"go-micro.dev/v6/agent"
)

func (s *session) interactAgent(ctx context.Context, name, message string) error {
	stream, err := agent.NewClient(name, s.cl).StreamAsk(ctx, message)
	if err != nil {
		return err
	}
	defer stream.Close()
	ctx = stream.Context()
	if s.display != nil {
		defer s.display.flush()
	}
	for {
		event, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if event.Type == agent.StreamEventDone {
			if s.display == nil {
				fmt.Fprintln(s.writer())
			}
			return nil
		}
		if event.Type == agent.StreamEventApproval {
			if s.display != nil {
				s.display.flush()
			}
			decision, err := s.promptApproval(ctx, event.ToolCall)
			if err != nil {
				if ctx.Err() != nil {
					return context.Cause(ctx)
				}
				return err
			}
			if err := stream.Approve(event.ApprovalID, decision.Status == agent.ApprovalApproved); err != nil {
				return err
			}
			continue
		}
		s.showEvent(event)
	}
}
