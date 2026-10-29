package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	pb "go-micro.dev/v6/agent/proto"
	"go-micro.dev/v6/broker"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/model"
	"go-micro.dev/v6/registry"
)

func TestClientConversations(t *testing.T) {
	fakeGen = func(_ context.Context, _ model.Options, req *model.Request) (*model.Response, error) {
		var previous []string
		for _, message := range req.Messages {
			previous = append(previous, fmt.Sprint(message.Content))
		}
		return &model.Response{Reply: strings.Join(previous, "|") + ":answer"}, nil
	}
	defer func() { fakeGen = nil }()
	reg := registry.NewMemoryRegistry()
	a := newTestAgent(Name("remote"), Address("127.0.0.1:0"), WithRegistry(reg), WithBroker(broker.NewMemoryBroker()))
	if _, err := a.startServer(); err != nil {
		t.Fatal(err)
	}
	defer a.Stop()
	remote := NewClient(a.Name(), client.NewClient(client.Registry(reg)))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	response, err := remote.Ask(WithSession(ctx, "one"), "private question")
	if err != nil {
		t.Fatal(err)
	}
	if response.Agent != a.Name() || response.RunID == "" {
		t.Fatalf("missing response metadata: %+v", response)
	}
	for _, id := range []string{"two", "one"} {
		stream, err := remote.StreamAsk(WithSession(ctx, id), "follow up")
		if err != nil {
			t.Fatal(err)
		}
		var done *Response
		for {
			event, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			if event.Type == StreamEventDone {
				done = event.Response
			}
		}
		_ = stream.Close()
		if done == nil || done.RunID == "" || done.Agent != a.Name() {
			t.Fatalf("missing completion: %+v", done)
		}
		if strings.Contains(done.Reply, "private question") != (id == "one") {
			t.Fatalf("session %s: %s", id, done.Reply)
		}
	}
}

type failingAgentClient struct {
	client.Client
	calls   int
	retries int
}

func (c *failingAgentClient) Call(_ context.Context, _ client.Request, _ interface{}, opts ...client.CallOption) error {
	options := client.CallOptions{Retries: 9}
	for _, opt := range opts {
		opt(&options)
	}
	c.retries = options.Retries
	c.calls++
	return errors.New("connection lost after tool execution")
}

func TestClientDoesNotRetryAsk(t *testing.T) {
	rpc := &failingAgentClient{Client: client.DefaultClient}
	_, err := NewClient("support", rpc).Ask(context.Background(), "update")
	if err == nil || rpc.calls != 1 || rpc.retries != 0 {
		t.Fatalf("calls=%d retries=%d err=%v", rpc.calls, rpc.retries, err)
	}
}

func TestClientResponseFields(t *testing.T) {
	response, err := decodeResponse(&pb.ChatResponse{Reply: "done", Agent: "support", RunId: "run", ParentId: "parent", ToolCalls: []*pb.ToolCall{{Id: "call", Name: "read", Input: `{"issue":42}`, Result: "found"}}})
	if err != nil {
		t.Fatal(err)
	}
	if response.ParentID != "parent" || response.RunID != "run" || len(response.ToolCalls) != 1 || response.ToolCalls[0].Input["issue"] != float64(42) || response.ToolCalls[0].Result != "found" {
		t.Fatalf("lost fields: %+v", response)
	}
	if _, err := decodeResponse(&pb.ChatResponse{ToolCalls: []*pb.ToolCall{{Input: "invalid"}}}); err == nil {
		t.Fatal("invalid tool input accepted")
	}
}

func TestClientCancellation(t *testing.T) {
	started, finished := make(chan struct{}), make(chan struct{})
	fakeGen = func(ctx context.Context, _ model.Options, _ *model.Request) (*model.Response, error) {
		close(started)
		<-ctx.Done()
		close(finished)
		return nil, ctx.Err()
	}
	defer func() { fakeGen = nil }()
	reg := registry.NewMemoryRegistry()
	a := newTestAgent(Name("cancellable"), Address("127.0.0.1:0"), WithRegistry(reg), WithBroker(broker.NewMemoryBroker()))
	if _, err := a.startServer(); err != nil {
		t.Fatal(err)
	}
	defer a.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream, err := NewClient(a.Name(), client.NewClient(client.Registry(reg))).StreamAsk(ctx, "wait")
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("model never started")
	}
	cancel()
	if _, err := stream.Recv(); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel returned %v", err)
	}
	select {
	case <-finished:
	case <-time.After(3 * time.Second):
		t.Fatal("remote model did not stop")
	}
}
