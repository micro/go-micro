package chat

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	pb "go-micro.dev/v6/agent/proto"
	"go-micro.dev/v6/broker"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/server"
)

type interactionTestHost struct{ disconnect <-chan struct{} }

func (h *interactionTestHost) Chat(_ context.Context, stream pb.AgentInteraction_ChatStream) error {
	if _, err := stream.Recv(); err != nil {
		return err
	}
	for _, event := range []*pb.InteractionEvent{
		{Type: "tool_start", ToolCall: &pb.ToolCall{Name: "workspace_write", Input: "{}"}},
		{Type: "approval", ApprovalId: "decision", ToolCall: &pb.ToolCall{Name: "workspace_write", Input: `{"path":"note.txt"}`}},
	} {
		if err := stream.Send(event); err != nil {
			return err
		}
	}
	if h.disconnect != nil {
		<-h.disconnect
		return nil
	}
	decision, err := stream.Recv()
	if err != nil {
		return err
	}
	text := "denied"
	if decision.ApprovalId == "decision" && decision.Approved {
		text = "saved"
	}
	for _, event := range []*pb.InteractionEvent{{Type: "tool_output", Text: "working\n"}, {Type: "tool_end", Result: text}, {Type: "token", Text: "finished"}, {Type: "done"}} {
		if err := stream.Send(event); err != nil {
			return err
		}
	}
	return nil
}

func TestRemoteApprovalUX(t *testing.T) {
	for _, mode := range []string{"approve", "deny", "disconnect"} {
		t.Run(mode, func(t *testing.T) {
			reg := registry.NewMemoryRegistry()
			host := server.NewServer(server.Name("ui-test"), server.Address("127.0.0.1:0"), server.Registry(reg), server.Broker(broker.NewMemoryBroker()))
			handler := &interactionTestHost{}
			disconnect := make(chan struct{})
			if mode == "disconnect" {
				handler.disconnect = disconnect
			}
			if err := pb.RegisterAgentInteractionHandler(host, handler); err != nil {
				t.Fatal(err)
			}
			if err := host.Start(); err != nil {
				t.Fatal(err)
			}
			defer host.Stop()
			var out bytes.Buffer
			s := &session{cl: client.NewClient(client.Registry(reg)), output: &out, interactiveUI: true, approvals: make(chan approvalRequest)}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			finished := make(chan error, 1)
			go func() { finished <- s.interactAgent(ctx, "ui-test", "write") }()
			select {
			case request := <-s.approvals:
				if mode == "disconnect" {
					close(disconnect)
				} else {
					request.answer <- mode == "approve"
				}
			case <-ctx.Done():
				t.Fatal("no approval prompt")
			}
			select {
			case err := <-finished:
				if (err != nil) != (mode == "disconnect") {
					t.Fatalf("result: %v", err)
				}
			case <-ctx.Done():
				t.Fatal("approval did not stop on disconnect")
			}
			text := out.String()
			for _, want := range []string{"Approve workspace_write?", "note.txt", "/approve or /deny"} {
				if !strings.Contains(text, want) {
					t.Fatalf("missing %q: %s", want, text)
				}
			}
			if mode != "disconnect" {
				want := "saved"
				if mode == "deny" {
					want = "denied"
				}
				for _, part := range []string{"working", want, "finished"} {
					if !strings.Contains(text, part) {
						t.Fatalf("missing %q: %s", part, text)
					}
				}
			}
		})
	}
}
