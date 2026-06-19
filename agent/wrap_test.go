package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
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

// A guardrail refusal carries a structured reason a wrapper can switch on,
// so reliability tooling (e.g. loop handling) needn't parse the message.
func TestWrapToolSeesRefusedReason(t *testing.T) {
	a := newTestAgent(Name("looper"), LoopLimit(2))
	h := a.toolHandler()

	var last ai.ToolResult
	for i := 0; i < 3; i++ {
		last = h(context.Background(), ai.ToolCall{ID: "x", Name: "demo_Svc_Do", Input: map[string]any{"q": "same"}})
	}
	if last.Refused != ai.RefusedLoop {
		t.Errorf("Refused = %q, want %q", last.Refused, ai.RefusedLoop)
	}
}

// ctxMock is a model that forwards the Generate context to the tool
// handler (as real providers do), so a wrapper can read ai.RunInfo.
type ctxMock struct{ opts ai.Options }

func (m *ctxMock) Init(opts ...ai.Option) error {
	for _, o := range opts {
		o(&m.opts)
	}
	return nil
}
func (m *ctxMock) Options() ai.Options { return m.opts }
func (m *ctxMock) String() string      { return "ctxmock" }
func (m *ctxMock) Stream(context.Context, *ai.Request, ...ai.GenerateOption) (ai.Stream, error) {
	return nil, fmt.Errorf("no stream")
}
func (m *ctxMock) Generate(ctx context.Context, _ *ai.Request, _ ...ai.GenerateOption) (*ai.Response, error) {
	if m.opts.ToolHandler != nil {
		m.opts.ToolHandler(ctx, ai.ToolCall{ID: "c1", Name: "demo_Svc_Do", Input: map[string]any{}})
	}
	return &ai.Response{Answer: "done"}, nil
}

// During an Ask, a wrapper sees RunInfo on the context: a correlation id
// for the run and the agent's name.
func TestWrapToolSeesRunInfo(t *testing.T) {
	ai.Register("ctxmock", func(opts ...ai.Option) ai.Model {
		m := &ctxMock{}
		_ = m.Init(opts...)
		return m
	})

	var got ai.RunInfo
	var ok bool
	a := New(
		Name("runner"),
		Provider("ctxmock"),
		WithRegistry(registry.NewMemoryRegistry()),
		WithStore(store.NewMemoryStore()),
		WrapTool(func(next ai.ToolHandler) ai.ToolHandler {
			return func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
				got, ok = ai.RunInfoFrom(ctx)
				return next(ctx, call)
			}
		}),
	)

	if _, err := a.Ask(context.Background(), "go"); err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if !ok {
		t.Fatal("wrapper did not see RunInfo on the context")
	}
	if got.Agent != "runner" {
		t.Errorf("RunInfo.Agent = %q, want runner", got.Agent)
	}
	if got.RunID == "" {
		t.Error("RunInfo.RunID is empty")
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
