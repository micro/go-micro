package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
)

type conformanceProvider struct {
	name  string
	model string
	key   string
	live  bool
}

func TestAgentProviderConformanceMatrix(t *testing.T) {
	providers := []conformanceProvider{
		{name: "fake"},
		{name: "openai", key: "OPENAI_API_KEY", model: "GO_MICRO_CONFORMANCE_OPENAI_MODEL", live: true},
		{name: "anthropic", key: "ANTHROPIC_API_KEY", model: "GO_MICRO_CONFORMANCE_ANTHROPIC_MODEL", live: true},
		{name: "atlascloud", key: "ATLASCLOUD_API_KEY", model: "GO_MICRO_CONFORMANCE_ATLASCLOUD_MODEL", live: true},
		{name: "gemini", key: "GEMINI_API_KEY", model: "GO_MICRO_CONFORMANCE_GEMINI_MODEL", live: true},
		{name: "groq", key: "GROQ_API_KEY", model: "GO_MICRO_CONFORMANCE_GROQ_MODEL", live: true},
		{name: "mistral", key: "MISTRAL_API_KEY", model: "GO_MICRO_CONFORMANCE_MISTRAL_MODEL", live: true},
		{name: "together", key: "TOGETHER_API_KEY", model: "GO_MICRO_CONFORMANCE_TOGETHER_MODEL", live: true},
	}

	selected := selectedConformanceProviders(os.Getenv("GO_MICRO_AGENT_CONFORMANCE_PROVIDERS"))
	for _, provider := range providers {
		provider := provider
		if len(selected) > 0 && !selected[provider.name] {
			continue
		}
		t.Run(provider.name, func(t *testing.T) {
			runAgentConformanceScenario(t, provider)
		})
	}
}

func selectedConformanceProviders(csv string) map[string]bool {
	out := map[string]bool{}
	for _, part := range strings.Split(csv, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out[part] = true
		}
	}
	return out
}

func runAgentConformanceScenario(t *testing.T, provider conformanceProvider) {
	t.Helper()
	if provider.live {
		if os.Getenv(provider.key) == "" {
			t.Skipf("%s not set; skipping live %s conformance", provider.key, provider.name)
		}
		if os.Getenv("GO_MICRO_AGENT_CONFORMANCE_LIVE") == "" {
			t.Skipf("GO_MICRO_AGENT_CONFORMANCE_LIVE not set; skipping live %s conformance", provider.name)
		}
	} else {
		fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
			if err := validateConformanceRequest(req, opts); err != nil {
				return nil, err
			}

			plan := opts.ToolHandler(ctx, ai.ToolCall{
				ID:   "fake-plan-1",
				Name: "plan",
				Input: map[string]any{"steps": []map[string]any{
					{"description": "call conformance_echo", "status": "pending"},
					{"description": "attempt guarded delegate", "status": "pending"},
				}},
			})
			echo := opts.ToolHandler(ctx, ai.ToolCall{
				ID:    "fake-call-1",
				Name:  "conformance_echo",
				Input: map[string]any{"value": "agent-conformance"},
			})
			delegate := opts.ToolHandler(ctx, ai.ToolCall{
				ID:    "fake-delegate-1",
				Name:  "delegate",
				Input: map[string]any{"task": "summarize the conformance marker", "to": "blocked-reviewer"},
			})
			if plan.Content == "" {
				return nil, errors.New("empty plan result")
			}
			if echo.Content == "" {
				return nil, errors.New("empty tool result")
			}
			if delegate.Refused != ai.RefusedApproval {
				return nil, fmt.Errorf("delegate refusal = %q, want %q", delegate.Refused, ai.RefusedApproval)
			}
			return &ai.Response{
				Reply:  "planned, called conformance_echo, and handled guarded delegate refusal",
				Answer: echo.Content + " " + delegate.Content,
				ToolCalls: []ai.ToolCall{
					{ID: "fake-plan-1", Name: "plan", Input: map[string]any{}},
					{ID: "fake-call-1", Name: "conformance_echo", Input: map[string]any{"value": "agent-conformance"}, Result: echo.Content},
					{ID: "fake-delegate-1", Name: "delegate", Input: map[string]any{"task": "summarize the conformance marker", "to": "blocked-reviewer"}, Error: delegate.Content},
				},
			}, nil
		}
		defer func() { fakeGen = nil }()
	}

	var sawTool bool
	var sawRunInfo bool
	var sawBlockedDelegate bool
	agentOpts := []Option{
		Name("conformance-" + provider.name),
		Provider(provider.name),
		APIKey(os.Getenv(provider.key)),
		Prompt(conformanceSystemPrompt(provider.name)),
		WithRegistry(registry.NewMemoryRegistry()),
		WithStore(store.NewMemoryStore()),
		WithMemory(NewInMemory(8)),
		ModelCallTimeout(45 * time.Second),
		ApproveTool(func(tool string, input map[string]any) (bool, string) {
			if tool == "delegate" {
				sawBlockedDelegate = true
				return false, "cross-provider conformance blocks delegate side effects"
			}
			return true, ""
		}),
		WithTool("conformance_echo", "Echo a conformance value and return a deterministic marker.", map[string]any{
			"value": map[string]any{"type": "string", "description": "value to echo"},
		}, func(ctx context.Context, input map[string]any) (string, error) {
			sawTool = true
			info, ok := ai.RunInfoFrom(ctx)
			if !ok {
				return "", errors.New("missing run info")
			}
			if info.RunID == "" || info.Agent != "conformance-"+provider.name {
				return "", fmt.Errorf("unexpected run info: %+v", info)
			}
			sawRunInfo = true
			if input["value"] != "agent-conformance" {
				return "", fmt.Errorf("unexpected value %v", input["value"])
			}
			return `{"marker":"agent-conformance-ok"}`, nil
		}),
	}
	if provider.model != "" {
		if model := os.Getenv(provider.model); model != "" {
			agentOpts = append(agentOpts, Model(model))
		}
	}

	a := New(agentOpts...)
	resp, err := askWithConformanceRetry(context.Background(), a, "Run the provider conformance check.", &sawTool, &sawBlockedDelegate)
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if resp.RunID == "" {
		t.Fatal("RunID is empty")
	}
	if resp.Agent != "conformance-"+provider.name {
		t.Fatalf("Agent = %q", resp.Agent)
	}
	if !sawTool {
		t.Fatal("provider did not request the conformance tool")
	}
	if !sawRunInfo {
		t.Fatal("tool did not receive RunInfo")
	}
	if !sawBlockedDelegate {
		t.Fatal("provider did not exercise the guarded delegate path")
	}
	if !strings.Contains(resp.Reply, "agent-conformance-ok") && !strings.Contains(resp.Reply, "agent-conformance") {
		t.Fatalf("reply %q does not include conformance marker", resp.Reply)
	}
}

func askWithConformanceRetry(ctx context.Context, a Agent, initialPrompt string, sawTool, sawBlockedDelegate *bool) (*Response, error) {
	const maxAttempts = 3
	prompt := initialPrompt
	var resp *Response
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		var err error
		resp, err = a.Ask(ctx, prompt)
		if err != nil {
			return nil, err
		}
		sawRequiredTool := sawTool == nil || *sawTool
		sawRequiredDelegate := sawBlockedDelegate == nil || *sawBlockedDelegate
		hasMarker := responseHasConformanceMarker(resp)
		if sawRequiredTool && sawRequiredDelegate && hasMarker {
			return resp, nil
		}
		if attempt == maxAttempts {
			break
		}
		prompt = nextConformanceRetryPrompt(sawRequiredTool, sawRequiredDelegate, hasMarker)
	}
	return resp, nil
}

func askWithConformanceToolRetry(ctx context.Context, a Agent, initialPrompt string, sawTool *bool) (*Response, error) {
	return askWithConformanceRetry(ctx, a, initialPrompt, sawTool, nil)
}

func conformanceSystemPrompt(provider string) string {
	prompt := "You are a conformance test agent. Create a short plan, use conformance_echo exactly once with input {\"value\":\"agent-conformance\"}, then attempt to delegate a summary to blocked-reviewer with input {\"task\":\"summarize the conformance marker\",\"to\":\"blocked-reviewer\"}. If the delegate is refused, explain the refusal and answer with the echo result."
	if provider == "atlascloud" {
		prompt += " AtlasCloud/minimax conformance note: the delegate attempt is mandatory after conformance_echo. If native tool_calls are unavailable, emit the delegate as <tool_call name=\"delegate\">{\"task\":\"summarize the conformance marker\",\"to\":\"blocked-reviewer\"}</tool_call> rather than answering in prose."
	}
	return prompt
}

func TestAgentProviderConformanceAtlasCloudPromptRequiresTaggedDelegateFallback(t *testing.T) {
	prompt := conformanceSystemPrompt("atlascloud")
	for _, want := range []string{
		"delegate attempt is mandatory",
		"<tool_call name=\"delegate\">",
		`{"task":"summarize the conformance marker","to":"blocked-reviewer"}`,
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("atlascloud conformance prompt %q missing %q", prompt, want)
		}
	}

	if strings.Contains(conformanceSystemPrompt("openai"), "AtlasCloud/minimax") {
		t.Fatal("non-AtlasCloud prompt should not include provider-specific fallback guidance")
	}
}

func nextConformanceRetryPrompt(sawTool, sawBlockedDelegate, hasMarker bool) string {
	switch {
	case !sawTool:
		return "The previous response did not call the required conformance_echo tool. Retry the same conformance check now: you must call conformance_echo exactly once with input {\"value\":\"agent-conformance\"} before any final answer, then include the tool result marker in the final answer."
	case !sawBlockedDelegate:
		return "The previous response called conformance_echo but did not attempt the required guarded delegation. Continue the same conformance check now: call delegate exactly once with input {\"task\":\"summarize the conformance marker\",\"to\":\"blocked-reviewer\"}; do not answer in prose until that delegate call has been attempted. If native tool_calls are unavailable, emit exactly <tool_call name=\"delegate\">{\"task\":\"summarize the conformance marker\",\"to\":\"blocked-reviewer\"}</tool_call>. The delegate is expected to be refused by policy; include that refusal and the agent-conformance marker in the final answer."
	case !hasMarker:
		return "The previous response completed the required tool calls but omitted the conformance marker. Continue the same conformance check now: do not call more tools; answer with the prior echo result marker agent-conformance-ok and mention the guarded delegate refusal."
	default:
		return "Retry the provider conformance check and include the agent-conformance marker in the final answer."
	}
}

func responseHasConformanceMarker(resp *Response) bool {
	if resp == nil {
		return false
	}
	return strings.Contains(resp.Reply, "agent-conformance-ok") || strings.Contains(resp.Reply, "agent-conformance")
}

func validateConformanceRequest(req *ai.Request, opts ai.Options) error {
	if req.Prompt == "" {
		return errors.New("missing prompt")
	}
	if len(req.Messages) == 0 || req.Messages[len(req.Messages)-1].Role != "user" {
		return fmt.Errorf("missing user history: %+v", req.Messages)
	}
	if len(req.Tools) == 0 {
		return errors.New("missing tools")
	}
	if opts.ToolHandler == nil {
		return errors.New("missing tool handler")
	}
	return nil
}

func TestAgentProviderConformanceFakeError(t *testing.T) {
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		return nil, errors.New("conformance provider failure")
	}
	defer func() { fakeGen = nil }()

	a := New(
		Name("conformance-error"),
		Provider("fake"),
		WithRegistry(registry.NewMemoryRegistry()),
		WithStore(store.NewMemoryStore()),
		WithMemory(NewInMemory(4)),
	)
	_, err := a.Ask(context.Background(), "fail deterministically")
	if err == nil || !strings.Contains(err.Error(), "conformance provider failure") {
		t.Fatalf("Ask error = %v, want conformance provider failure", err)
	}
}

func TestAgentProviderConformanceRetriesMissingTool(t *testing.T) {
	var attempts int
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		attempts++
		if err := validateConformanceRequest(req, opts); err != nil {
			return nil, err
		}
		if attempts == 1 {
			return &ai.Response{Reply: "I can confirm agent-conformance in prose only."}, nil
		}
		echo := opts.ToolHandler(ctx, ai.ToolCall{
			ID:    "fake-call-1",
			Name:  "conformance_echo",
			Input: map[string]any{"value": "agent-conformance"},
		})
		return &ai.Response{
			Reply:  "called conformance_echo",
			Answer: echo.Content,
			ToolCalls: []ai.ToolCall{
				{ID: "fake-call-1", Name: "conformance_echo", Input: map[string]any{"value": "agent-conformance"}, Result: echo.Content},
			},
		}, nil
	}
	defer func() { fakeGen = nil }()

	var sawTool bool
	a := New(
		Name("conformance-retry"),
		Provider("fake"),
		WithRegistry(registry.NewMemoryRegistry()),
		WithStore(store.NewMemoryStore()),
		WithMemory(NewInMemory(4)),
		WithTool("conformance_echo", "Echo a conformance value.", map[string]any{
			"value": map[string]any{"type": "string"},
		}, func(ctx context.Context, input map[string]any) (string, error) {
			sawTool = true
			return `{"marker":"agent-conformance-ok"}`, nil
		}),
	)

	resp, err := askWithConformanceToolRetry(context.Background(), a, "Run the provider conformance check.", &sawTool)
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want retry after missing tool", attempts)
	}
	if !sawTool {
		t.Fatal("retry did not execute conformance_echo")
	}
	if !strings.Contains(resp.Reply, "agent-conformance-ok") {
		t.Fatalf("Reply = %q, want tool result marker", resp.Reply)
	}
}

func TestAgentProviderConformanceRetriesMissingMarker(t *testing.T) {
	var attempts int
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		attempts++
		if err := validateConformanceRequest(req, opts); err != nil {
			return nil, err
		}
		if attempts == 1 {
			return &ai.Response{Reply: "called conformance_echo and handled guarded delegate refusal without the required marker"}, nil
		}
		return &ai.Response{Reply: "agent-conformance-ok after guarded delegate refusal"}, nil
	}
	defer func() { fakeGen = nil }()

	sawTool := true
	sawBlockedDelegate := true
	a := New(
		Name("conformance-retry-marker"),
		Provider("fake"),
		WithRegistry(registry.NewMemoryRegistry()),
		WithStore(store.NewMemoryStore()),
		WithMemory(NewInMemory(4)),
		WithTool("conformance_echo", "Echo a conformance value.", map[string]any{
			"value": map[string]any{"type": "string"},
		}, func(ctx context.Context, input map[string]any) (string, error) {
			return `{"marker":"agent-conformance-ok"}`, nil
		}),
	)

	resp, err := askWithConformanceRetry(context.Background(), a, "Run the provider conformance check.", &sawTool, &sawBlockedDelegate)
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want retry after missing marker", attempts)
	}
	if !strings.Contains(resp.Reply, "agent-conformance-ok") {
		t.Fatalf("Reply = %q, want conformance marker", resp.Reply)
	}
}

func TestAgentProviderConformanceRetriesMissingDelegate(t *testing.T) {
	var attempts int
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		attempts++
		if err := validateConformanceRequest(req, opts); err != nil {
			return nil, err
		}
		echo := opts.ToolHandler(ctx, ai.ToolCall{
			ID:    "fake-call-1",
			Name:  "conformance_echo",
			Input: map[string]any{"value": "agent-conformance"},
		})
		if attempts == 1 {
			return &ai.Response{
				Reply:  "called conformance_echo but skipped delegate",
				Answer: echo.Content,
				ToolCalls: []ai.ToolCall{
					{ID: "fake-call-1", Name: "conformance_echo", Input: map[string]any{"value": "agent-conformance"}, Result: echo.Content},
				},
			}, nil
		}
		delegate := opts.ToolHandler(ctx, ai.ToolCall{
			ID:    "fake-delegate-1",
			Name:  "delegate",
			Input: map[string]any{"task": "summarize the conformance marker", "to": "blocked-reviewer"},
		})
		return &ai.Response{
			Reply:  "called conformance_echo and handled guarded delegate refusal",
			Answer: echo.Content + " " + delegate.Content,
			ToolCalls: []ai.ToolCall{
				{ID: "fake-call-1", Name: "conformance_echo", Input: map[string]any{"value": "agent-conformance"}, Result: echo.Content},
				{ID: "fake-delegate-1", Name: "delegate", Input: map[string]any{"task": "summarize the conformance marker", "to": "blocked-reviewer"}, Error: delegate.Content},
			},
		}, nil
	}
	defer func() { fakeGen = nil }()

	var sawTool bool
	var sawBlockedDelegate bool
	a := New(
		Name("conformance-retry-delegate"),
		Provider("fake"),
		WithRegistry(registry.NewMemoryRegistry()),
		WithStore(store.NewMemoryStore()),
		WithMemory(NewInMemory(4)),
		ApproveTool(func(tool string, input map[string]any) (bool, string) {
			if tool == "delegate" {
				sawBlockedDelegate = true
				return false, "cross-provider conformance blocks delegate side effects"
			}
			return true, ""
		}),
		WithTool("conformance_echo", "Echo a conformance value.", map[string]any{
			"value": map[string]any{"type": "string"},
		}, func(ctx context.Context, input map[string]any) (string, error) {
			sawTool = true
			return `{"marker":"agent-conformance-ok"}`, nil
		}),
	)

	resp, err := askWithConformanceRetry(context.Background(), a, "Run the provider conformance check.", &sawTool, &sawBlockedDelegate)
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want retry after missing delegate", attempts)
	}
	if !sawBlockedDelegate {
		t.Fatal("retry did not attempt guarded delegate")
	}
	if !strings.Contains(resp.Reply, "agent-conformance-ok") && !strings.Contains(resp.Reply, "agent-conformance") {
		t.Fatalf("Reply = %q, want conformance marker", resp.Reply)
	}
}

func TestAgentExecutesProviderTextToolCallFallback(t *testing.T) {
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		if opts.ToolHandler == nil {
			return nil, errors.New("missing tool handler")
		}
		return &ai.Response{
			Reply: `{"name":"conformance_echo","input":{"value":"agent-conformance"}}`,
		}, nil
	}
	defer func() { fakeGen = nil }()

	var sawTool bool
	a := New(
		Name("conformance-text-tool"),
		Provider("fake"),
		WithRegistry(registry.NewMemoryRegistry()),
		WithStore(store.NewMemoryStore()),
		WithMemory(NewInMemory(4)),
		WithTool("conformance_echo", "Echo a conformance value.", map[string]any{
			"value": map[string]any{"type": "string"},
		}, func(ctx context.Context, input map[string]any) (string, error) {
			sawTool = true
			if input["value"] != "agent-conformance" {
				return "", fmt.Errorf("unexpected value %v", input["value"])
			}
			return `{"marker":"agent-conformance-ok"}`, nil
		}),
	)

	resp, err := a.Ask(context.Background(), "Run the text tool call fallback.")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if !sawTool {
		t.Fatal("text tool call fallback did not execute the tool")
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "conformance_echo" {
		t.Fatalf("ToolCalls = %+v, want conformance_echo", resp.ToolCalls)
	}
	if !strings.Contains(resp.Reply, "agent-conformance-ok") {
		t.Fatalf("Reply = %q, want tool result marker", resp.Reply)
	}
	if strings.Contains(resp.Reply, `"name":"conformance_echo"`) {
		t.Fatalf("Reply = %q, want tool result instead of raw JSON", resp.Reply)
	}
}
