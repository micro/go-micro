package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
)

func TestStoreMemoryPersists(t *testing.T) {
	st := store.NewMemoryStore()
	m := NewMemory(st, "agent/x/history", 10)
	m.Add("user", "hello")
	m.Add("assistant", "hi there")

	// A fresh memory over the same store/key restores the conversation.
	reloaded := NewMemory(st, "agent/x/history", 10)
	if got := len(reloaded.Messages()); got != 2 {
		t.Fatalf("restored %d messages, want 2", got)
	}
}

func TestInMemoryNotPersisted(t *testing.T) {
	m := NewInMemory(10)
	m.Add("user", "x")
	if got := len(m.Messages()); got != 1 {
		t.Fatalf("got %d messages, want 1", got)
	}
	if got := len(NewInMemory(10).Messages()); got != 0 {
		t.Errorf("a separate in-memory should be empty, got %d", got)
	}
}

func TestMemoryClearPersists(t *testing.T) {
	st := store.NewMemoryStore()
	m := NewMemory(st, "agent/y/history", 10)
	m.Add("user", "x")
	m.Clear()
	if got := len(m.Messages()); got != 0 {
		t.Errorf("after Clear got %d messages, want 0", got)
	}
	if got := len(NewMemory(st, "agent/y/history", 10).Messages()); got != 0 {
		t.Errorf("cleared state should persist, reload got %d", got)
	}
}

func TestWithMemoryUsed(t *testing.T) {
	custom := NewInMemory(5)
	a := New(
		Name("z"),
		Provider("fake"),
		WithRegistry(registry.NewMemoryRegistry()),
		WithStore(store.NewMemoryStore()),
		WithMemory(custom),
	).(*agentImpl)
	a.setup()
	if a.mem != custom {
		t.Error("WithMemory should make the agent use the supplied memory")
	}
}

// A custom tool is offered to the model and dispatched to its handler.
func TestWithToolExposedAndDispatched(t *testing.T) {
	var got map[string]any
	a := newTestAgent(Name("calc-agent"),
		WithTool("calc", "adds two numbers",
			map[string]any{
				"a": map[string]any{"type": "number"},
				"b": map[string]any{"type": "number"},
			},
			func(ctx context.Context, input map[string]any) (string, error) {
				got = input
				return `{"sum":3}`, nil
			}))

	tools, err := a.discoverTools()
	if err != nil {
		t.Fatalf("discoverTools: %v", err)
	}
	found := false
	for _, tl := range tools {
		if tl.Name == "calc" {
			found = true
		}
	}
	if !found {
		t.Fatal("custom tool 'calc' was not offered to the model")
	}

	content := toolContent(a.toolHandler(), "calc", map[string]any{"a": 1.0, "b": 2.0})
	if got == nil {
		t.Fatal("custom tool handler was not called")
	}
	if !strings.Contains(content, "sum") {
		t.Errorf("custom tool result not returned: %q", content)
	}
}

// A custom tool returning an error surfaces it to the model.
func TestWithToolError(t *testing.T) {
	a := newTestAgent(Name("err-agent"),
		WithTool("boom", "always fails", nil,
			func(ctx context.Context, input map[string]any) (string, error) {
				return "", errors.New("kaboom")
			}))

	content := toolContent(a.toolHandler(), "boom", nil)
	if !strings.Contains(content, "kaboom") {
		t.Errorf("tool error not surfaced: %q", content)
	}
}
