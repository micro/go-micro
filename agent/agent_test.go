package agent

import (
	"testing"
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
	if opts.Broker == nil {
		t.Error("Broker should default to DefaultBroker")
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
