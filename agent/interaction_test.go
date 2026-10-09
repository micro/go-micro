package agent

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	pb "go-micro.dev/v6/agent/proto"
	"go-micro.dev/v6/broker"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/model"
	"go-micro.dev/v6/registry"
)

func TestInteractionApproval(t *testing.T) {
	for _, mode := range []string{"approve", "deny", "disconnect", "wrong decision", "host denies"} {
		t.Run(mode, func(t *testing.T) {
			var executed atomic.Int32
			finished := make(chan struct{})
			fakeGen = func(ctx context.Context, opts model.Options, _ *model.Request) (*model.Response, error) {
				defer close(finished)
				opts.ToolHandler(ctx, model.ToolCall{ID: "call", Name: "write", Input: map[string]any{"path": "note"}})
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				return &model.Response{Reply: "done"}, nil
			}
			defer func() { fakeGen = nil }()
			approval := RequestApproval
			if mode == "host denies" {
				approval = func(context.Context, model.ToolCall) (ApprovalDecision, error) {
					return ApprovalDecision{Status: ApprovalDenied}, nil
				}
			}
			reg := registry.NewMemoryRegistry()
			a := newTestAgent(Name("interactive"), Address("127.0.0.1:0"), WithRegistry(reg), WithBroker(broker.NewMemoryBroker()), WithApproval(approval),
				WithTool("write", "write", nil, func(ctx context.Context, _ map[string]any) (string, error) {
					executed.Add(1)
					ToolOutput(ctx, "writing")
					return "saved", nil
				}))
			if _, err := a.startServer(); err != nil {
				t.Fatal(err)
			}
			defer a.Stop()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			stream, err := pb.NewAgentInteractionService(a.Name(), client.NewClient(client.Registry(reg))).Chat(WithSession(ctx, "session"))
			if err != nil {
				t.Fatal(err)
			}
			defer stream.Close()
			if err := stream.Send(&pb.InteractionRequest{Message: "write"}); err != nil {
				t.Fatal(err)
			}
			seen := map[string]bool{}
			for {
				event, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				seen[event.Type] = true
				if event.Type == string(StreamEventApproval) {
					if mode == "host denies" {
						t.Fatal("host denial overridden")
					}
					if executed.Load() != 0 {
						t.Fatal("executed before approval")
					}
					if event.ToolCall.Name != "write" || event.ApprovalId == "" {
						t.Fatalf("approval: %+v", event)
					}
					if mode == "disconnect" {
						stream.Close()
						break
					}
					id := event.ApprovalId
					if mode == "wrong decision" {
						id = "wrong"
					}
					if err := stream.Send(&pb.InteractionRequest{ApprovalId: id, Approved: mode == "approve"}); err != nil {
						t.Fatal(err)
					}
					if mode == "wrong decision" {
						break
					}
				}
				if event.Type == string(StreamEventDone) {
					if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
						t.Fatalf("end of stream: %v", err)
					}
					break
				}
			}
			select {
			case <-finished:
			case <-ctx.Done():
				t.Fatal("agent did not finish after decision or disconnect")
			}
			want := int32(0)
			if mode == "approve" {
				want = 1
			}
			if executed.Load() != want {
				t.Fatalf("executed %d, want %d", executed.Load(), want)
			}
			if mode == "approve" && (!seen["tool_start"] || !seen["tool_output"] || !seen["tool_end"] || !seen["done"]) {
				t.Fatalf("events: %v", seen)
			}
		})
	}
}

func TestRequestApprovalWithoutConnection(t *testing.T) {
	d, err := RequestApproval(context.Background(), model.ToolCall{Name: "write"})
	if err != nil || d.Status != ApprovalDenied {
		t.Fatalf("%+v %v", d, err)
	}
}

func TestInteractionReturnsProviderError(t *testing.T) {
	const message = "provider unavailable for this request"
	fakeGen = func(context.Context, model.Options, *model.Request) (*model.Response, error) {
		return nil, errors.New(message)
	}
	defer func() { fakeGen = nil }()
	reg := registry.NewMemoryRegistry()
	a := newTestAgent(Name("error-host"), Address("127.0.0.1:0"), WithRegistry(reg), WithBroker(broker.NewMemoryBroker()))
	if _, err := a.startServer(); err != nil {
		t.Fatal(err)
	}
	defer a.Stop()
	api := pb.NewAgentInteractionService(a.Name(), client.NewClient(client.Registry(reg)))
	for i := 0; i < 2; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		stream, err := api.Chat(WithSession(ctx, "error-session"))
		if err != nil {
			cancel()
			t.Fatal(err)
		}
		if err := stream.Send(&pb.InteractionRequest{Message: "hello"}); err != nil {
			stream.Close()
			cancel()
			t.Fatal(err)
		}
		_, err = stream.Recv()
		stream.Close()
		cancel()
		if err == nil || !strings.Contains(err.Error(), message) {
			t.Fatalf("provider error lost: %v", err)
		}
	}
}
