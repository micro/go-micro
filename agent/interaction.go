package agent

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"

	"github.com/google/uuid"
	pb "go-micro.dev/v6/agent/proto"
	"go-micro.dev/v6/model"
)

type eventKey struct{}
type approvalKey struct{}

// ToolOutput sends incremental tool output to the current agent stream.
// Outside a stream it does nothing.
func ToolOutput(ctx context.Context, text string) {
	emitEvent(ctx, &StreamEvent{Type: StreamEventToolOutput, Token: text})
}

func emitEvent(ctx context.Context, event *StreamEvent) bool {
	send, ok := ctx.Value(eventKey{}).(func(*StreamEvent) bool)
	return ok && send(event)
}

// RequestApproval asks the connected interactive client to approve a tool call.
// Hosts opt in through WithApproval; this never overrides a host's policy.
// Without an interactive connection the action is denied. Disconnecting cancels
// the request. Hosts remain responsible for authenticating clients.
func RequestApproval(ctx context.Context, call model.ToolCall) (ApprovalDecision, error) {
	if approve, ok := ctx.Value(approvalKey{}).(ApprovalFunc); ok {
		return approve(ctx, call)
	}
	return ApprovalDecision{Status: ApprovalDenied, Reason: "This action needs approval from a connected chat."}, nil
}

type interactionHandler struct{ agent *agentImpl }

func (h *interactionHandler) Chat(ctx context.Context, stream pb.AgentInteraction_ChatStream) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	first, err := stream.Recv()
	if err != nil {
		return err
	}
	if first.ApprovalId != "" {
		return errors.New("agent: expected a chat message")
	}
	conversation, release, err := h.agent.rpcSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	var mu sync.Mutex
	var approvalMu sync.Mutex
	var pendingID string
	var pending chan bool
	go func() {
		defer cancel()
		for {
			request, err := stream.Recv()
			if err != nil {
				return
			}
			mu.Lock()
			if request.Message != "" || pending == nil || request.ApprovalId != pendingID {
				mu.Unlock()
				return
			}
			pending <- request.Approved
			pending = nil
			pendingID = ""
			mu.Unlock()
		}
	}()
	approve := ApprovalFunc(func(callCtx context.Context, call model.ToolCall) (ApprovalDecision, error) {
		approvalMu.Lock()
		defer approvalMu.Unlock()
		if err := callCtx.Err(); err != nil {
			return ApprovalDecision{}, err
		}
		id := uuid.New().String()
		answer := make(chan bool, 1)
		mu.Lock()
		pendingID, pending = id, answer
		mu.Unlock()
		defer func() { mu.Lock(); pendingID, pending = "", nil; mu.Unlock() }()
		if !emitEvent(callCtx, &StreamEvent{Type: StreamEventApproval, ToolCall: call, ApprovalID: id}) {
			return ApprovalDecision{}, errors.New("agent: approval connection closed")
		}
		select {
		case allowed := <-answer:
			if err := callCtx.Err(); err != nil {
				return ApprovalDecision{}, err
			}
			if allowed {
				return ApprovalDecision{Status: ApprovalApproved}, nil
			}
			return ApprovalDecision{Status: ApprovalDenied, Reason: "Denied by user"}, nil
		case <-callCtx.Done():
			return ApprovalDecision{}, callCtx.Err()
		}
	})
	ctx = context.WithValue(ctx, approvalKey{}, approve)
	events, err := conversation.StreamAsk(ctx, first.Message)
	if err != nil {
		return err
	}
	defer events.Close()
	for {
		event, err := events.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		data, err := json.Marshal(event.ToolCall.Input)
		if err != nil {
			return err
		}
		out := &pb.InteractionEvent{Type: string(event.Type), Text: event.Token, ApprovalId: event.ApprovalID, Result: event.Result.Content,
			ToolCall: &pb.ToolCall{Id: event.ToolCall.ID, Name: event.ToolCall.Name, Input: string(data)}}
		if event.Response != nil {
			out.Response = &pb.ChatResponse{Reply: event.Response.Reply, Agent: event.Response.Agent, RunId: event.Response.RunID, ParentId: event.Response.ParentID}
			for _, call := range event.Response.ToolCalls {
				input, err := json.Marshal(call.Input)
				if err != nil {
					return err
				}
				out.Response.ToolCalls = append(out.Response.ToolCalls, &pb.ToolCall{Id: call.ID, Name: call.Name, Input: string(input), Result: call.Result})
			}
		}
		if err := stream.Send(out); err != nil {
			cancel()
			return err
		}
	}
}
