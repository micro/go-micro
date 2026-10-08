package agent

import (
	"context"
	"testing"

	pb "go-micro.dev/v6/agent/proto"
	"go-micro.dev/v6/model"
	"go-micro.dev/v6/store"
)

func TestSessionsPersistAndIsolate(t *testing.T) {
	dir := t.TempDir()
	var history []model.Message
	fakeGen = func(_ context.Context, _ model.Options, req *model.Request) (*model.Response, error) {
		history = req.Messages
		return &model.Response{Reply: "answer"}, nil
	}
	defer func() { fakeGen = nil }()
	ask := func(id, prompt string) {
		t.Helper()
		// Reopen the file store and agent, as after a process restart.
		state := store.NewFileStore(store.DirOption(dir))
		defer state.Close()
		a := newTestAgent(Name("shared"), WithStore(state), Session(id))
		if _, err := a.Ask(context.Background(), prompt); err != nil {
			t.Fatal(err)
		}
	}
	ask("one", "private one")
	ask("two", "private two")
	if len(history) != 0 {
		t.Fatalf("session two saw history: %+v", history)
	}
	ask("one", "continue")
	if len(history) != 2 || history[0].Content != "private one" {
		t.Fatalf("lost session one: %+v", history)
	}
}

func TestRPCSessionIsolation(t *testing.T) {
	var history []model.Message
	fakeGen = func(_ context.Context, _ model.Options, req *model.Request) (*model.Response, error) {
		history = req.Messages
		return &model.Response{Reply: "answer"}, nil
	}
	defer func() { fakeGen = nil }()
	a := newTestAgent(Name("remote"), WithStore(store.NewMemoryStore()))
	for _, id := range []string{"one", "two", "one"} {
		if err := a.Chat(WithSession(context.Background(), id), &pb.ChatRequest{Message: id}, &pb.ChatResponse{}); err != nil {
			t.Fatal(err)
		}
		if id == "two" && len(history) != 0 {
			t.Fatal("RPC conversation leaked")
		}
	}
	if len(history) != 2 {
		t.Fatalf("history=%+v", history)
	}
}

type sessionStream struct {
	pb.Agent_StreamChatStream
	response string
}

func (s *sessionStream) Recv() (*pb.ChatRequest, error) {
	return &pb.ChatRequest{Message: "streamed"}, nil
}
func (s *sessionStream) Send(r *pb.ChatResponse) error { s.response += r.Reply; return nil }

func TestRPCStreamUsesSessionHistory(t *testing.T) {
	var history []model.Message
	fakeGen = func(_ context.Context, _ model.Options, req *model.Request) (*model.Response, error) {
		history = req.Messages
		return &model.Response{Reply: "answer"}, nil
	}
	defer func() { fakeGen = nil }()
	a := newTestAgent(Name("stream-session"), WithStore(store.NewMemoryStore()))
	stream := &sessionStream{}
	ctx := WithSession(context.Background(), "one")
	if err := a.StreamChat(ctx, stream); err != nil {
		t.Fatal(err)
	}
	if stream.response != "answer" {
		t.Fatalf("reply=%q", stream.response)
	}
	if err := a.Chat(ctx, &pb.ChatRequest{Message: "next"}, &pb.ChatResponse{}); err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || history[0].Content != "streamed" {
		t.Fatalf("history=%+v", history)
	}
}
