package agent

import (
	"context"
	"testing"

	pb "go-micro.dev/v6/agent/proto"
	"go-micro.dev/v6/ai"
)

func TestNew(t *testing.T) {
	a := New(
		Name("test-agent"),
		Services("task", "project"),
		Prompt("You manage tasks."),
		Provider("anthropic"),
	)

	if a.Name() != "test-agent" {
		t.Errorf("Name() = %q, want %q", a.Name(), "test-agent")
	}

	opts := a.Options()
	if opts.Provider != "anthropic" {
		t.Errorf("Provider = %q, want %q", opts.Provider, "anthropic")
	}
	if len(opts.Services) != 2 {
		t.Fatalf("Services = %v, want 2 items", opts.Services)
	}
	if opts.Services[0] != "task" || opts.Services[1] != "project" {
		t.Errorf("Services = %v, want [task project]", opts.Services)
	}
	if opts.Prompt != "You manage tasks." {
		t.Errorf("Prompt = %q, want %q", opts.Prompt, "You manage tasks.")
	}
	if opts.HistoryLimit != 50 {
		t.Errorf("HistoryLimit = %d, want 50", opts.HistoryLimit)
	}
}

func TestChatResponseIncludesRunIDs(t *testing.T) {
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		return &ai.Response{Reply: "ok"}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("chat-run"))
	var rsp pb.ChatResponse
	if err := a.Chat(context.Background(), &pb.ChatRequest{Message: "hello"}, &rsp); err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if rsp.RunId == "" {
		t.Fatal("Chat response RunId is empty")
	}
	if rsp.Agent != "chat-run" {
		t.Errorf("Agent = %q, want chat-run", rsp.Agent)
	}
	if rsp.ParentId != "" {
		t.Errorf("ParentId = %q, want empty", rsp.ParentId)
	}
}

func TestChatRequestParentIDPropagatesToResponse(t *testing.T) {
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		info, ok := ai.RunInfoFrom(ctx)
		if !ok {
			t.Fatal("RunInfo missing from model context")
		}
		if info.ParentID != "flow-run-123" {
			t.Fatalf("RunInfo.ParentID = %q, want flow-run-123", info.ParentID)
		}
		return &ai.Response{Reply: "ok"}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("chat-child"))
	var rsp pb.ChatResponse
	if err := a.Chat(context.Background(), &pb.ChatRequest{Message: "hello", ParentId: "flow-run-123"}, &rsp); err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if rsp.ParentId != "flow-run-123" {
		t.Errorf("ParentId = %q, want flow-run-123", rsp.ParentId)
	}
}

func TestBuildPrompt(t *testing.T) {
	// Custom prompt
	a := New(Name("test"), Prompt("custom prompt")).(*agentImpl)
	if got := a.buildPrompt(); got != "custom prompt" {
		t.Errorf("buildPrompt() = %q, want %q", got, "custom prompt")
	}

	// Auto-generated prompt with services
	a = New(Name("test"), Services("task", "project")).(*agentImpl)
	got := a.buildPrompt()
	if got == "" {
		t.Error("buildPrompt() returned empty")
	}
	if !contains(got, "task") || !contains(got, "project") {
		t.Errorf("buildPrompt() = %q, should mention services", got)
	}

	// Auto-generated prompt without services
	a = New(Name("test")).(*agentImpl)
	got = a.buildPrompt()
	if !contains(got, "test") {
		t.Errorf("buildPrompt() = %q, should mention agent name", got)
	}
}

func TestDefaults(t *testing.T) {
	a := New(Name("test"))
	opts := a.Options()

	if opts.Registry == nil {
		t.Error("Registry should default to DefaultRegistry")
	}
	if opts.Client == nil {
		t.Error("Client should default to DefaultClient")
	}
	if opts.Store == nil {
		t.Error("Store should default to DefaultStore")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
