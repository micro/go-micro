package agent

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
)

func TestBuiltinTools(t *testing.T) {
	tools := builtinTools()
	if len(tools) != 3 {
		t.Fatalf("builtinTools() = %d tools, want 3", len(tools))
	}
	names := map[string]bool{}
	for _, tl := range tools {
		names[tl.Name] = true
	}
	if !names[toolPlan] || !names[toolDelegate] || !names[toolHumanInput] {
		t.Errorf("builtin tools = %v, want plan, request_input, and delegate", names)
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
	content := a.handlePlan(ai.ToolCall{Name: "plan", Input: steps}).Content
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

func TestHandlePlanPreservesCompletedSteps(t *testing.T) {
	mem := store.NewMemoryStore()
	a := New(Name("planner"), WithStore(mem)).(*agentImpl)

	a.handlePlan(ai.ToolCall{Name: "plan", Input: map[string]any{
		"steps": []any{
			map[string]any{"task": "create Design task", "status": "done"},
			map[string]any{"task": "Delegate readiness notification to comms agent", "status": "done"},
		},
	}})

	res := a.handlePlan(ai.ToolCall{Name: "plan", Input: map[string]any{
		"steps": []any{
			map[string]any{"task": "create Design task", "status": "done"},
			map[string]any{"task": "  delegate   readiness notification TO comms agent  ", "status": "in_progress"},
			map[string]any{"task": "write summary", "status": "pending"},
		},
	}})
	if res.Content == "" {
		t.Fatal("handlePlan returned empty content")
	}
	if unfinished := a.unfinishedPlanSteps(); len(unfinished) != 1 || unfinished[0] != "write summary" {
		t.Fatalf("unfinished plan steps = %v, want only write summary", unfinished)
	}
}

func TestHandlePlanPreservesCompletedLaunchReadinessNotification(t *testing.T) {
	mem := store.NewMemoryStore()
	a := New(Name("planner"), WithStore(mem)).(*agentImpl)

	a.handlePlan(ai.ToolCall{Name: toolPlan, Input: map[string]any{
		"steps": []any{
			map[string]any{"task": "notify owner via comms", "status": "done"},
		},
	}})

	a.handlePlan(ai.ToolCall{Name: toolPlan, Input: map[string]any{
		"steps": []any{
			map[string]any{"task": "Delegate launch readiness notification for owner@acme.com to comms agent", "status": "in_progress"},
		},
	}})

	if unfinished := a.unfinishedPlanSteps(); len(unfinished) != 0 {
		t.Fatalf("unfinished plan steps = %v, want launch readiness notification preserved as done", unfinished)
	}
}

func TestPlanShowsInPrompt(t *testing.T) {
	mem := store.NewMemoryStore()
	a := New(Name("planner"), Prompt("base prompt"), WithStore(mem)).(*agentImpl)

	if got := a.buildPrompt(); got != "base prompt" {
		t.Errorf("buildPrompt() with no plan = %q, want %q", got, "base prompt")
	}

	a.handlePlan(ai.ToolCall{Name: "plan", Input: map[string]any{"steps": []any{map[string]any{"task": "do it", "status": "pending"}}}})

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

	if len(tools) != 3 {
		t.Fatalf("Builtins() returned %d tools, want 3", len(tools))
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
	scoped := store.Scope(mem, "agent", "chat")
	if recs, err := scoped.Read(planKey); err != nil || len(recs) == 0 {
		t.Errorf("plan not persisted in the agent's scoped store: err=%v recs=%d", err, len(recs))
	}
}

func TestDelegateResultCacheReusesLaunchReadinessParaphrases(t *testing.T) {
	mem := store.NewMemoryStore()
	a := New(Name("planner"), WithStore(mem)).(*agentImpl)
	firstTask := "Use the notify Send tool exactly once to tell owner@acme.com: The launch plan is ready."
	first := a.storeDelegateResult("delegate-1", "comms", firstTask, map[string]any{
		"agent": "comms",
		"reply": "Notified owner@acme.com.",
	})
	if first.Content == "" {
		t.Fatal("storeDelegateResult returned empty content")
	}

	replayedTasks := []string{
		"Notify the plan owner at owner @ acme.com that launch readiness is prepared and complete.",
		"Tell owner at acme dot com the launch readiness notification was sent and the plan is done.",
	}
	for i, replayedTask := range replayedTasks {
		cached, ok := a.cachedDelegateResult("delegate-replay", "  COMMS  ", replayedTask)
		if !ok {
			t.Fatalf("cachedDelegateResult missed equivalent launch-readiness delegate replay %d", i)
		}
		if cached.ID != "delegate-replay" {
			t.Fatalf("cached result ID = %q, want replay call ID", cached.ID)
		}
		if !containsStr(cached.Content, "Notified owner@acme.com") {
			t.Fatalf("cached result content = %q, want original delegate reply", cached.Content)
		}
	}
}

func TestDelegateInFlightReplaysShareFirstResult(t *testing.T) {
	a := New(Name("planner"), WithStore(store.NewMemoryStore())).(*agentImpl)
	key := delegateResultKey("comms", "Notify owner@acme.com that the launch plan is ready")
	if _, joined := a.joinDelegateCall(context.Background(), "delegate-1", key); joined {
		t.Fatal("first delegate call unexpectedly joined an existing in-flight call")
	}

	var wg sync.WaitGroup
	wg.Add(1)
	results := make(chan ai.ToolResult, 1)
	go func() {
		defer wg.Done()
		res, joined := a.joinDelegateCall(context.Background(), "delegate-2", key)
		if !joined {
			t.Error("replayed delegate call did not join the in-flight call")
			return
		}
		results <- res
	}()

	select {
	case res := <-results:
		t.Fatalf("replayed delegate returned before first call finished: %+v", res)
	case <-time.After(25 * time.Millisecond):
	}

	first := ai.ToolResult{ID: "delegate-1", Content: `{"reply":"Notified owner@acme.com."}`}
	a.finishDelegateCall(key, first)
	wg.Wait()
	replayed := <-results
	if replayed.ID != "delegate-2" {
		t.Fatalf("replayed result ID = %q, want delegate-2", replayed.ID)
	}
	if replayed.Content != first.Content {
		t.Fatalf("replayed content = %q, want %q", replayed.Content, first.Content)
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

func TestPlanWrapBlocksDelegationUntilPriorPlanStepsFinish(t *testing.T) {
	mem := store.NewMemoryStore()
	a := New(Name("planner"), WithStore(mem)).(*agentImpl)
	a.handlePlan(ai.ToolCall{Name: toolPlan, Input: map[string]any{
		"steps": []any{
			map[string]any{"task": "Create Design task", "status": "pending"},
			map[string]any{"task": "Create Build task", "status": "pending"},
			map[string]any{"task": "Create Ship task", "status": "pending"},
			map[string]any{"task": "Delegate readiness notification to comms agent", "status": "pending"},
		},
	}})

	called := false
	handle := a.planWrap(func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		called = true
		return ai.ToolResult{ID: call.ID, Content: "ok"}
	})

	res := handle(context.Background(), ai.ToolCall{ID: "delegate-1", Name: toolDelegate, Input: map[string]any{"to": "comms"}})
	if called {
		t.Fatal("delegate handler was called before prior task plan steps completed")
	}
	if res.Refused == "" {
		t.Fatalf("delegate result was not refused: %+v", res)
	}
	if got := res.Content; !containsStr(got, "Create Design task") || !containsStr(got, "Create Ship task") {
		t.Fatalf("delegate refusal content = %q, want prior unfinished task steps", got)
	}

	for _, id := range []string{"add-design", "add-build", "add-ship"} {
		_ = handle(context.Background(), ai.ToolCall{ID: id, Name: "task.Add", Input: map[string]any{"title": id}})
	}
	called = false
	res = handle(context.Background(), ai.ToolCall{ID: "delegate-2", Name: toolDelegate, Input: map[string]any{"to": "comms"}})
	if !called {
		t.Fatal("delegate handler was not called after prior task plan steps completed")
	}
	if res.Refused != "" {
		t.Fatalf("delegate result refused after prior task steps completed: %+v", res)
	}
}
