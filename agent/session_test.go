package agent

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"
	"time"

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
	state := store.NewFileStore(store.DirOption(dir))
	messages, err := LoadHistory(state, "shared", "one")
	if err != nil || len(messages) != 2 || messages[0].Content != "private one" {
		t.Fatalf("stored history=%+v err=%v", messages, err)
	}
	if err := state.Close(); err != nil {
		t.Fatal(err)
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

func TestSessionRunsRemainDiscoverable(t *testing.T) {
	st := store.NewMemoryStore()
	a := newTestAgent(Name("sessions"), WithStore(st), Session("one"))
	a.recordRunEvent(RunEvent{RunID: "first", Kind: "done"})
	a.opts.Session = "two"
	a.recordRunEvent(RunEvent{RunID: "second", Kind: "done"})
	runs, err := ListRunSummaries(st, "sessions")
	if err != nil || len(runs) != 2 {
		t.Fatalf("runs=%+v err=%v", runs, err)
	}
	runs, err = ListRunSummariesWithOptions(st, "sessions", RunListOptions{Session: "one"})
	if err != nil || len(runs) != 1 || runs[0].RunID != "first" {
		t.Fatalf("filtered runs=%+v err=%v", runs, err)
	}
}

func TestSessionLocksAreIndependent(t *testing.T) {
	a := &agentImpl{}
	release, err := a.lockSession(context.Background(), "one")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := a.lockSession(ctx, "one"); err != context.Canceled {
		t.Fatalf("waiting request: %v", err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	other, err := a.lockSession(ctx, "two")
	if err != nil {
		t.Fatalf("independent session blocked: %v", err)
	}
	other()
	release()
	if len(a.sessions) != 0 {
		t.Fatal("unused locks retained")
	}
}

type customSessionMemory struct {
	Memory
	sessions map[string]Memory
}

func (m *customSessionMemory) Session(id string) (Memory, error) { return m.sessions[id], nil }

func TestRPCCustomSessionMemory(t *testing.T) {
	one, two := NewMemory(store.NewMemoryStore(), "one", 100), NewMemory(store.NewMemoryStore(), "two", 100)
	backend := &customSessionMemory{Memory: NewMemory(store.NewMemoryStore(), "default", 100), sessions: map[string]Memory{"one": one, "two": two}}
	a := newTestAgent(Name("custom"), WithStore(store.NewMemoryStore()), WithMemory(backend))
	for id, expected := range backend.sessions {
		child, release, err := a.rpcSession(WithSession(context.Background(), id))
		if err != nil {
			t.Fatal(err)
		}
		if child.opts.Memory != expected {
			t.Fatal("custom session memory ignored")
		}
		release()
	}
}

func TestSessionNamespaceCannotOverlapAgentName(t *testing.T) {
	for _, files := range []bool{false, true} {
		t.Run(fmt.Sprint(files), func(t *testing.T) {
			st := store.NewMemoryStore()
			if files {
				st = store.NewFileStore(store.DirOption(t.TempDir()))
			}
			defer st.Close()
			name, id := "foo", "conversation"
			other := fmt.Sprintf("%s-session-%x", name, sha256.Sum256([]byte(id)))
			session := sessionStore(st, name, id)
			ordinary := sessionStore(st, other, "")
			for _, key := range []string{"history", "plan"} {
				if err := session.Write(&store.Record{Key: key, Value: []byte("session")}); err != nil {
					t.Fatal(err)
				}
				if _, err := ordinary.Read(key); !errors.Is(err, store.ErrNotFound) {
					t.Fatalf("%s leaked to ordinary agent: %v", key, err)
				}
				if err := ordinary.Write(&store.Record{Key: key, Value: []byte("ordinary")}); err != nil {
					t.Fatal(err)
				}
				records, err := session.Read(key)
				if err != nil || len(records) != 1 || string(records[0].Value) != "session" {
					t.Fatalf("%s overwritten: %+v %v", key, records, err)
				}
			}
		})
	}
}
