package agent

import (
	"encoding/json"
	"testing"

	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/store"
)

func TestBuiltinTools(t *testing.T) {
	tools := builtinTools()
	if len(tools) != 2 {
		t.Fatalf("builtinTools() = %d tools, want 2", len(tools))
	}
	names := map[string]bool{}
	for _, tl := range tools {
		names[tl.Name] = true
	}
	if !names[toolPlan] || !names[toolDelegate] {
		t.Errorf("builtin tools = %v, want plan and delegate", names)
	}
}

func TestHandlePlanPersists(t *testing.T) {
	mem := store.NewMemoryStore()
	a := New(Name("planner"), WithStore(mem)).(*agentImpl)

	steps := map[string]any{
		"steps": []any{
			map[string]any{"task": "gather requirements", "status": "done"},
			map[string]any{"task": "write code", "status": "in_progress"},
		},
	}
	_, content := a.handlePlan(steps)
	if content == "" {
		t.Fatal("handlePlan returned empty content")
	}

	// The plan must be retrievable from memory.
	got := a.loadPlan()
	if got == "" {
		t.Fatal("loadPlan() returned empty after handlePlan")
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("stored plan is not valid JSON: %v", err)
	}
	if _, ok := decoded["steps"]; !ok {
		t.Errorf("stored plan missing steps: %s", got)
	}
}

func TestPlanShowsInPrompt(t *testing.T) {
	mem := store.NewMemoryStore()
	a := New(Name("planner"), Prompt("base prompt"), WithStore(mem)).(*agentImpl)

	if got := a.buildPrompt(); got != "base prompt" {
		t.Errorf("buildPrompt() with no plan = %q, want %q", got, "base prompt")
	}

	a.handlePlan(map[string]any{"steps": []any{map[string]any{"task": "do it", "status": "pending"}}})

	got := a.buildPrompt()
	if got == "base prompt" {
		t.Error("buildPrompt() should include the plan once one is saved")
	}
	if !containsStr(got, "do it") {
		t.Errorf("buildPrompt() = %q, should contain the saved plan", got)
	}
}

func TestDiscoverToolsIncludesBuiltins(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	a := New(Name("a"), WithRegistry(reg), WithStore(store.NewMemoryStore())).(*agentImpl)
	a.setup()

	tools, err := a.discoverTools()
	if err != nil {
		t.Fatalf("discoverTools: %v", err)
	}
	// No services registered, so the only tools should be the builtins.
	if len(tools) != len(builtinTools()) {
		t.Fatalf("discoverTools() = %d tools, want %d builtins", len(tools), len(builtinTools()))
	}
}

func TestEphemeralAgentHasNoBuiltins(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	a := New(Name("a.sub"), WithRegistry(reg), WithStore(store.NewMemoryStore())).(*agentImpl)
	a.ephemeral = true
	a.setup()

	tools, err := a.discoverTools()
	if err != nil {
		t.Fatalf("discoverTools: %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("ephemeral agent discoverTools() = %d tools, want 0", len(tools))
	}
}

func TestBuiltinsAccessor(t *testing.T) {
	mem := store.NewMemoryStore()
	tools, handle := Builtins(
		Name("chat"),
		WithStore(mem),
		WithRegistry(registry.NewMemoryRegistry()),
	)

	if len(tools) != 2 {
		t.Fatalf("Builtins() returned %d tools, want 2", len(tools))
	}

	// A name that isn't a built-in falls through (ok == false).
	if _, _, ok := handle("not_a_builtin", nil); ok {
		t.Error("handle(non-builtin) ok = true, want false")
	}

	// plan is handled and persisted under the configured name.
	_, content, ok := handle(toolPlan, map[string]any{
		"steps": []any{map[string]any{"task": "x", "status": "pending"}},
	})
	if !ok {
		t.Fatal("handle(plan) ok = false, want true")
	}
	if content == "" {
		t.Fatal("handle(plan) returned empty content")
	}
	if recs, err := mem.Read("agent/chat/plan"); err != nil || len(recs) == 0 {
		t.Errorf("plan not persisted under agent/chat/plan: err=%v recs=%d", err, len(recs))
	}
}

func TestIsAgent(t *testing.T) {
	reg := registry.NewMemoryRegistry()

	// A plain service.
	if err := reg.Register(&registry.Service{
		Name:  "task",
		Nodes: []*registry.Node{{Id: "task-1", Address: "127.0.0.1:0"}},
	}); err != nil {
		t.Fatalf("register service: %v", err)
	}
	// An agent (advertises type=agent).
	if err := reg.Register(&registry.Service{
		Name:     "task-mgr",
		Metadata: map[string]string{"type": "agent"},
		Nodes:    []*registry.Node{{Id: "task-mgr-1", Address: "127.0.0.1:0"}},
	}); err != nil {
		t.Fatalf("register agent: %v", err)
	}

	a := New(Name("root"), WithRegistry(reg)).(*agentImpl)

	if a.isAgent("task") {
		t.Error("isAgent(task) = true, want false (plain service)")
	}
	if !a.isAgent("task-mgr") {
		t.Error("isAgent(task-mgr) = false, want true (agent)")
	}
	if a.isAgent("nonexistent") {
		t.Error("isAgent(nonexistent) = true, want false")
	}
}
