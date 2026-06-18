package agent

import (
	"context"
	"strings"
	"testing"

	"go-micro.dev/v6/ai"
)

// A registered wrapper runs around every tool call and can observe and
// modify the result.
func TestWrapToolWraps(t *testing.T) {
	var saw string
	wrap := func(next ai.ToolHandler) ai.ToolHandler {
		return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
			saw = call.Name
			res := next(ctx, call)
			res.Content = "wrapped:" + res.Content
			return res
		}
	}

	a := newTestAgent(Name("wrapped"), WrapTool(wrap))
	content := toolContent(a.toolHandler(), "demo_Svc_Do", map[string]any{})

	if saw != "demo_Svc_Do" {
		t.Errorf("wrapper saw %q, want demo_Svc_Do", saw)
	}
	if !strings.HasPrefix(content, "wrapped:") {
		t.Errorf("wrapper did not modify the result; got %q", content)
	}
}

// Multiple wrappers compose outermost-first: the first registered wrapper
// is the outer layer, so it runs first on the way in and last on the way
// out.
func TestWrapToolOrder(t *testing.T) {
	var order []string
	mk := func(tag string) ai.ToolWrapper {
		return func(next ai.ToolHandler) ai.ToolHandler {
			return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
				order = append(order, "in:"+tag)
				res := next(ctx, call)
				order = append(order, "out:"+tag)
				return res
			}
		}
	}

	a := newTestAgent(Name("ordered"), WrapTool(mk("a"), mk("b")))
	toolContent(a.toolHandler(), "demo_Svc_Do", map[string]any{})

	want := "in:a in:b out:b out:a"
	if got := strings.Join(order, " "); got != want {
		t.Errorf("wrapper order = %q, want %q", got, want)
	}
}

// Wrappers run outside the built-in guardrails, so they observe a refused
// call and its refusal result rather than being short-circuited.
func TestWrapToolSeesGuardrailRefusal(t *testing.T) {
	var sawResult string
	wrap := func(next ai.ToolHandler) ai.ToolHandler {
		return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
			res := next(ctx, call)
			sawResult = res.Content
			return res
		}
	}

	a := newTestAgent(Name("gated-wrap"),
		ApproveTool(func(tool string, input map[string]any) (bool, string) {
			return false, "denied"
		}),
		WrapTool(wrap),
	)
	toolContent(a.toolHandler(), "demo_Svc_Do", map[string]any{})

	if !strings.Contains(sawResult, "not approved") {
		t.Errorf("wrapper should observe the guardrail refusal; got %q", sawResult)
	}
}

// call.Scan decodes a tool call's input into a typed struct.
func TestToolCallScan(t *testing.T) {
	call := ai.ToolCall{Input: map[string]any{"query": "hello", "limit": 5}}
	var args struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := call.Scan(&args); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if args.Query != "hello" || args.Limit != 5 {
		t.Errorf("Scan decoded %+v, want {hello 5}", args)
	}
}
