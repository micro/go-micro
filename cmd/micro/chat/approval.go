package chat

import (
	"context"
	"encoding/json"
	"fmt"

	"go-micro.dev/v6/agent"
	"go-micro.dev/v6/model"
)

type approvalRequest struct {
	ctx    context.Context
	answer chan bool
}

func (s *session) approve(ctx context.Context, call model.ToolCall) (agent.ApprovalDecision, error) {
	if s.yes || call.Name == "workspace_read" || call.Name == "workspace_search" || call.Name == "workspace_skill" || call.Name == "plan" {
		return agent.ApprovalDecision{Status: agent.ApprovalApproved}, nil
	}
	if !s.interactiveUI {
		return agent.ApprovalDecision{Status: agent.ApprovalDenied, Reason: "This action requires interactive approval. Use micro chat or explicitly allow tool actions with --yes."}, nil
	}
	s.approvalMu.Lock()
	defer s.approvalMu.Unlock()
	if err := ctx.Err(); err != nil {
		return agent.ApprovalDecision{}, err
	}
	request := approvalRequest{ctx: ctx, answer: make(chan bool, 1)}
	data, _ := json.MarshalIndent(call.Input, "", "  ")
	fmt.Fprintf(s.writer(), "\nApprove %s?\n%s\n/approve or /deny\n", call.Name, data)
	select {
	case s.approvals <- request:
	case <-ctx.Done():
		return agent.ApprovalDecision{}, ctx.Err()
	}

	select {
	case approved := <-request.answer:
		if approved {
			return agent.ApprovalDecision{Status: agent.ApprovalApproved}, nil
		}
		return agent.ApprovalDecision{Status: agent.ApprovalDenied, Reason: "Denied by user"}, nil
	case <-ctx.Done():
		return agent.ApprovalDecision{}, ctx.Err()
	}
}

func (s *session) answerApproval(approved bool) {
	for {
		select {
		case request := <-s.approvals:
			if request.ctx.Err() != nil {
				continue
			}
			request.answer <- approved
			return
		default:
			fmt.Fprintln(s.writer(), "No tool action is awaiting approval.")
			return
		}
	}
}
