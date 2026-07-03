// A2A stream fallback harness.
//
// It exercises the gateway boundary that fronts an agent over A2A. The agent is
// configured with tools and memory, but its model streaming path deliberately
// reports ai.ErrStreamingUnsupported; the A2A gateway must fall back to the
// normal Ask path and still complete the same tool-calling run.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"go-micro.dev/v6/agent"
	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/gateway/a2a"
	"go-micro.dev/v6/internal/harness/harnessutil"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
)

type mockModel struct{ opts ai.Options }

func newMock(opts ...ai.Option) ai.Model {
	m := &mockModel{}
	_ = m.Init(opts...)
	return m
}

func (m *mockModel) Init(opts ...ai.Option) error {
	for _, o := range opts {
		o(&m.opts)
	}
	return nil
}
func (m *mockModel) Options() ai.Options { return m.opts }
func (m *mockModel) String() string      { return "mock" }
func (m *mockModel) Stream(context.Context, *ai.Request, ...ai.GenerateOption) (ai.Stream, error) {
	return nil, ai.ErrStreamingUnsupported
}
func (m *mockModel) Generate(ctx context.Context, req *ai.Request, _ ...ai.GenerateOption) (*ai.Response, error) {
	if req.Prompt == "" {
		return nil, errors.New("missing prompt")
	}
	if len(req.Messages) == 0 || req.Messages[len(req.Messages)-1].Role != "user" {
		return nil, fmt.Errorf("missing user history: %+v", req.Messages)
	}
	if len(req.Tools) == 0 || m.opts.ToolHandler == nil {
		return nil, errors.New("missing tools or tool handler")
	}
	res := m.opts.ToolHandler(ctx, ai.ToolCall{ID: "a2a-fallback-call", Name: "fallback_echo", Input: map[string]any{"value": "a2a-fallback"}})
	if res.Content == "" {
		return nil, errors.New("empty tool result")
	}
	return &ai.Response{Reply: "fallback completed", Answer: res.Content, ToolCalls: []ai.ToolCall{{ID: "a2a-fallback-call", Name: "fallback_echo", Input: map[string]any{"value": "a2a-fallback"}, Result: res.Content}}}, nil
}

func providerKey(provider string) string {
	if v := os.Getenv("MICRO_AI_API_KEY"); v != "" {
		return v
	}
	env := map[string]string{
		"anthropic": "ANTHROPIC_API_KEY", "openai": "OPENAI_API_KEY",
		"gemini": "GEMINI_API_KEY", "groq": "GROQ_API_KEY", "mistral": "MISTRAL_API_KEY",
		"together": "TOGETHER_API_KEY", "atlascloud": "ATLASCLOUD_API_KEY",
	}[provider]
	return os.Getenv(env)
}

func main() {
	provider := flag.String("provider", "mock", "LLM provider: mock (default), anthropic, openai, ...")
	flag.Parse()

	apiKey := ""
	if *provider == "mock" {
		ai.Register("mock", newMock)
	} else {
		apiKey = providerKey(*provider)
		if apiKey == "" {
			fmt.Printf("no API key for provider %q — set MICRO_AI_API_KEY or the provider's key env\n", *provider)
			return
		}
	}

	fmt.Printf("\n\033[1mA2A streaming fallback conformance (provider: %s)\033[0m\n", *provider)
	reg := registry.NewMemoryRegistry()
	st := store.NewMemoryStore()
	var sawTool, sawRunInfo bool
	agentOpts := []agent.Option{
		agent.Name("a2a-fallback"),
		agent.Provider(*provider),
		agent.APIKey(apiKey),
		agent.Prompt("Use fallback_echo exactly once with value a2a-fallback, then answer with the tool result."),
		agent.WithRegistry(reg),
		agent.WithStore(st),
		agent.WithMemory(agent.NewInMemory(8)),
		agent.ModelCallTimeout(45 * time.Second),
		agent.WithTool("fallback_echo", "Echo the A2A fallback marker.", map[string]any{
			"value": map[string]any{"type": "string", "description": "value to echo"},
		}, func(ctx context.Context, input map[string]any) (string, error) {
			sawTool = true
			info, ok := ai.RunInfoFrom(ctx)
			if !ok || info.RunID == "" || info.Agent != "a2a-fallback" {
				return "", fmt.Errorf("unexpected run info: %+v", info)
			}
			sawRunInfo = true
			if input["value"] != "a2a-fallback" {
				return "", fmt.Errorf("unexpected value %v", input["value"])
			}
			return `{"marker":"a2a-fallback-ok"}`, nil
		}),
	}
	agentOpts = append(agentOpts, harnessutil.AgentOptions(*provider)...)
	ag := agent.New(agentOpts...)

	card := a2a.Card("a2a-fallback", "http://example.invalid/a2a-fallback", "", nil)
	handler := a2a.NewAgentStreamHandler(card, func(ctx context.Context, text string) (string, error) {
		resp, err := ag.Ask(ctx, text)
		if err != nil {
			return "", err
		}
		return resp.Reply, nil
	}, ag.Stream)

	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"message/stream","params":{"message":{"role":"user","parts":[{"kind":"text","text":"Run the A2A fallback conformance check."}],"kind":"message"}}}`)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	res := rr.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		fmt.Fprintf(os.Stderr, "unexpected status %d: %s\n", res.StatusCode, b)
		os.Exit(1)
	}
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		fmt.Fprintf(os.Stderr, "content-type = %q, want text/event-stream\n", ct)
		os.Exit(1)
	}
	summary, err := readSSESummary(res.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if summary.State != "completed" {
		fmt.Fprintf(os.Stderr, "stream final state = %q, want completed; payload: %s\n", summary.State, summary.Payload)
		os.Exit(1)
	}
	if !summary.HasArtifactText {
		fmt.Fprintf(os.Stderr, "stream completed without artifact text: %s\n", summary.Payload)
		os.Exit(1)
	}
	if !sawTool || !sawRunInfo {
		fmt.Fprintf(os.Stderr, "tool=%v runInfo=%v\n", sawTool, sawRunInfo)
		os.Exit(1)
	}
	fmt.Println("\n\033[32m✓ A2A message/stream fell back to Ask and preserved tool/run metadata\033[0m")
}

type streamSummary struct {
	Payload         string
	State           string
	HasArtifactText bool
}

func readSSESummary(r io.Reader) (streamSummary, error) {
	scanner := bufio.NewScanner(r)
	var event strings.Builder
	var summary streamSummary
	seen := false
	flush := func() error {
		data := strings.TrimSpace(event.String())
		event.Reset()
		if data == "" {
			return nil
		}
		var envelope struct {
			Result struct {
				Status struct {
					State string `json:"state"`
				} `json:"status"`
				Artifacts []struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"artifacts"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(data), &envelope); err != nil {
			return fmt.Errorf("SSE data event is not JSON: %s", data)
		}
		seen = true
		summary.Payload += data + "\n"
		if envelope.Result.Status.State != "" {
			summary.State = envelope.Result.Status.State
		}
		for _, artifact := range envelope.Result.Artifacts {
			for _, part := range artifact.Parts {
				if strings.TrimSpace(part.Text) != "" {
					summary.HasArtifactText = true
				}
			}
		}
		return nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			if err := flush(); err != nil {
				return streamSummary{}, err
			}
			continue
		}
		data, ok := strings.CutPrefix(line, "data:")
		if !ok {
			continue
		}
		if event.Len() > 0 {
			event.WriteByte('\n')
		}
		event.WriteString(strings.TrimSpace(data))
	}
	if err := scanner.Err(); err != nil {
		return streamSummary{}, err
	}
	if err := flush(); err != nil {
		return streamSummary{}, err
	}
	if !seen {
		return streamSummary{}, errors.New("no SSE data received")
	}
	return summary, nil
}
