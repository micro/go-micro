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

	for _, provider := range providers {
		provider := provider
		t.Run(provider.name, func(t *testing.T) {
			runAgentConformanceScenario(t, provider)
		})
	}
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
			if req.Prompt == "" {
				return nil, errors.New("missing prompt")
			}
			if len(req.Messages) == 0 || req.Messages[len(req.Messages)-1].Role != "user" {
				return nil, fmt.Errorf("missing user history: %+v", req.Messages)
			}
			if len(req.Tools) == 0 {
				return nil, errors.New("missing tools")
			}
			if opts.ToolHandler == nil {
				return nil, errors.New("missing tool handler")
			}
			res := opts.ToolHandler(ctx, ai.ToolCall{
				ID:    "fake-call-1",
				Name:  "conformance_echo",
				Input: map[string]any{"value": "agent-conformance"},
			})
			if res.Content == "" {
				return nil, errors.New("empty tool result")
			}
			return &ai.Response{
				Reply:     "used conformance_echo",
				Answer:    res.Content,
				ToolCalls: []ai.ToolCall{{ID: "fake-call-1", Name: "conformance_echo", Input: map[string]any{"value": "agent-conformance"}, Result: res.Content}},
			}, nil
		}
		defer func() { fakeGen = nil }()
	}

	var sawTool bool
	var sawRunInfo bool
	agentOpts := []Option{
		Name("conformance-" + provider.name),
		Provider(provider.name),
		APIKey(os.Getenv(provider.key)),
		Prompt("You are a conformance test agent. Use the conformance_echo tool exactly once with input {\"value\":\"agent-conformance\"}, then answer with the tool result."),
		WithRegistry(registry.NewMemoryRegistry()),
		WithStore(store.NewMemoryStore()),
		WithMemory(NewInMemory(8)),
		ModelCallTimeout(45 * time.Second),
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
	resp, err := a.Ask(context.Background(), "Run the provider conformance check.")
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
	if !strings.Contains(resp.Reply, "agent-conformance-ok") && !strings.Contains(resp.Reply, "agent-conformance") {
		t.Fatalf("reply %q does not include conformance marker", resp.Reply)
	}
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
