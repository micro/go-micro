package chat

import (
	"context"
	"testing"

	"go-micro.dev/v6/model"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
)

func TestSessionDiscoveryChecksEveryNode(t *testing.T) {
	for _, supported := range []bool{false, true} {
		reg := registry.NewMemoryRegistry()
		second := map[string]string{}
		if supported {
			second["sessions"] = "v1"
		}
		err := reg.Register(&registry.Service{Name: "assistant", Metadata: map[string]string{"type": "agent", "sessions": "v1"}, Nodes: []*registry.Node{
			{Id: "new", Address: "127.0.0.1:1", Metadata: map[string]string{"sessions": "v1"}},
			{Id: "other", Address: "127.0.0.1:2", Metadata: second},
		}})
		if err != nil {
			t.Fatal(err)
		}
		s := &session{reg: reg}
		if !s.discoverAgents() || s.agents["assistant"].Sessions != supported {
			t.Fatalf("supported=%v agents=%+v", supported, s.agents)
		}
	}
}

func TestLocalSessionSeparatesProjects(t *testing.T) {
	var history []model.Message
	s := testSession(t, func(_ context.Context, r *model.Request, _ model.Options) (*model.Response, error) {
		history = r.Messages
		return &model.Response{Reply: "answer"}, nil
	})
	s.state = store.NewMemoryStore()
	s.id = "same-name"
	first, second := t.TempDir(), t.TempDir()
	s.project = first
	firstKey := s.localSessionID()
	if _, err := s.developmentAgent().Ask(context.Background(), "first project secret"); err != nil {
		t.Fatal(err)
	}
	s.project = second
	s.reset()
	if firstKey == s.localSessionID() {
		t.Fatal("projects share a state key")
	}
	if _, err := s.developmentAgent().Ask(context.Background(), "second project"); err != nil {
		t.Fatal(err)
	}
	if len(history) != 0 {
		t.Fatalf("project history leaked: %+v", history)
	}
	s.project = first
	s.reset()
	if _, err := s.developmentAgent().Ask(context.Background(), "continue"); err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || history[0].Content != "first project secret" {
		t.Fatalf("project history lost: %+v", history)
	}
}
