package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"go-micro.dev/v6/model"
	"go-micro.dev/v6/store"
)

func TestRecallStaysInLocalProject(t *testing.T) {
	var recalled []recalledMessage
	var toolAvailable bool
	s := testSession(t, func(ctx context.Context, r *model.Request, o model.Options) (*model.Response, error) {
		toolAvailable = false
		for _, tool := range r.Tools {
			if tool.Name == "memory_search" {
				toolAvailable = true
			}
		}
		if r.Prompt == "recall" {
			result := o.ToolHandler(ctx, model.ToolCall{ID: "search", Name: "memory_search", Input: map[string]any{"query": "postgres"}})
			if err := json.Unmarshal([]byte(result.Content), &recalled); err != nil {
				t.Errorf("tool result: %q %v", result.Content, err)
			}
		}
		return &model.Response{Reply: "noted"}, nil
	})
	s.state = store.NewMemoryStore()
	first, second := t.TempDir(), t.TempDir()
	for _, entry := range []struct{ project, id, text string }{
		{first, "earlier", "postgres decision in this project"},
		{second, "earlier", "postgres private to another project"},
	} {
		s.project, s.id = entry.project, entry.id
		s.reset()
		if err := s.remember(entry.text); err != nil {
			t.Fatal(err)
		}
		if _, err := s.developmentAgent().Ask(context.Background(), entry.text); err != nil {
			t.Fatal(err)
		}
	}
	s.project, s.id = first, "current"
	s.reset()
	if _, err := s.developmentAgent().Ask(context.Background(), "recall"); err != nil {
		t.Fatal(err)
	}
	if !toolAvailable || len(recalled) != 1 || recalled[0].Session != "earlier" || recalled[0].Content != "postgres decision in this project" {
		t.Fatalf("recall crossed project boundary or lost source: %+v", recalled)
	}
	var output bytes.Buffer
	s.output = &output
	if err := s.memoryCommand(context.Background(), "/search --all postgres"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "earlier") || strings.Contains(output.String(), "private to another project") {
		t.Fatalf("search output: %s", output.String())
	}
	if err := s.memoryCommand(context.Background(), "/search --all"); err == nil {
		t.Fatal("empty search accepted")
	}
	s.agents = map[string]agentInfo{"remote": {History: true}}
	if err := s.memoryCommand(context.Background(), "/search --all postgres"); err == nil {
		t.Fatal("remote search used local scope")
	}
	s.agents = nil
	s.hosting = true
	s.reset()
	if _, err := s.developmentAgent().Ask(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	if toolAvailable {
		t.Fatal("host inherited local cross-conversation access")
	}
}

func TestRecallExcerptKeepsMatchingPassage(t *testing.T) {
	text := strings.Repeat("前", 3000) + "postgres migration choice" + strings.Repeat("後", 3000)
	excerpt := recallExcerpt(text, "postgres migration")
	if !strings.Contains(excerpt, "postgres migration choice") || len([]rune(excerpt)) > 2000 {
		t.Fatalf("matching passage lost or excerpt unbounded: %q", excerpt)
	}
}
