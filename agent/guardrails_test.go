package agent

import (
	"context"
	"strings"
	"testing"

	"go-micro.dev/v5/ai"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/store"
)

// toolContent runs a tool call through a handler and returns the content
// shown to the model — the part these tests assert on.
func toolContent(h ai.ToolHandler, name string, input map[string]any) string {
	return h(context.Background(), ai.ToolCall{Name: name, Input: input}).Content
}

// MaxSteps refuses tool calls once the per-Ask limit is exceeded; plan
// is bookkeeping and is never counted.
func TestMaxStepsStopsActions(t *testing.T) {
	a := newTestAgent(Name("limited"), MaxSteps(2))

	h := a.toolHandler()

	// plan must not consume a step.
	a.steps = 0
	toolContent(h, toolPlan, map[string]any{"steps": []any{}})
	if a.steps != 0 {
		t.Fatalf("plan consumed a step: steps=%d", a.steps)
	}

	// First two actions are allowed (they fall through to RPC, which
	// fails harmlessly — we only care they weren't refused by the limit).
	for i := 1; i <= 2; i++ {
		content := toolContent(h, "demo_Svc_Do", map[string]any{})
		if strings.Contains(content, "step limit") {
			t.Fatalf("action %d wrongly hit the step limit", i)
		}
	}

	// Third action exceeds MaxSteps(2) and must be refused.
	content := toolContent(h, "demo_Svc_Do", map[string]any{})
	if !strings.Contains(content, "step limit") {
		t.Errorf("third action should hit the step limit; got %q", content)
	}
}

// ApproveTool blocks an action when the hook denies it, and the denial
// reason is surfaced to the model.
func TestApproveToolBlocks(t *testing.T) {
	var sawTool string
	a := newTestAgent(Name("gated"),
		ApproveTool(func(tool string, input map[string]any) (bool, string) {
			sawTool = tool
			return false, "needs sign-off"
		}),
	)

	content := toolContent(a.toolHandler(), "demo_Svc_Do", map[string]any{})
	if sawTool != "demo_Svc_Do" {
		t.Errorf("approver saw %q, want demo_Svc_Do", sawTool)
	}
	if !strings.Contains(content, "not approved") || !strings.Contains(content, "needs sign-off") {
		t.Errorf("blocked call should surface the reason; got %q", content)
	}
}

// A denying approver must not gate the internal plan tool.
func TestApproveToolDoesNotGatePlan(t *testing.T) {
	mem := store.NewMemoryStore()
	a := New(
		Name("gated"),
		Provider("fake"),
		WithRegistry(registry.NewMemoryRegistry()),
		WithStore(mem),
		ApproveTool(func(tool string, input map[string]any) (bool, string) {
			return false, "deny everything"
		}),
	).(*agentImpl)
	a.setup()

	content := toolContent(a.toolHandler(), toolPlan, map[string]any{
		"steps": []any{map[string]any{"task": "x", "status": "pending"}},
	})
	if strings.Contains(content, "not approved") {
		t.Errorf("plan must not be gated by ApproveTool; got %q", content)
	}
	if recs, _ := store.Scope(mem, "agent", "gated").Read(planKey); len(recs) == 0 {
		t.Error("plan should have been persisted despite the denying approver")
	}
}
