// A2A streaming harness.
//
// It exercises the default, no-secret agent streaming path across the
// services → agents → A2A boundary: an A2A message/stream request invokes an
// agent StreamAsk turn, the agent executes a tool, and the gateway emits
// working SSE task updates before the completed final answer.
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
	if len(req.Tools) == 0 || m.opts.ToolHandler == nil {
		return nil, errors.New("missing tools or tool handler")
	}
	res := m.opts.ToolHandler(ctx, ai.ToolCall{ID: "a2a-stream-call", Name: "stream_echo", Input: map[string]any{"value": "a2a-stream"}})
	if res.Content == "" {
		return nil, errors.New("empty tool result")
	}
	return &ai.Response{
		Reply:  "streaming completed",
		Answer: res.Content,
		ToolCalls: []ai.ToolCall{{
			ID: "a2a-stream-call", Name: "stream_echo", Input: map[string]any{"value": "a2a-stream"}, Result: res.Content,
		}},
	}, nil
}

type agentStreamAdapter struct{ stream agent.AgentStream }

func (s agentStreamAdapter) Recv() (*ai.Response, error) {
	for {
		event, err := s.stream.Recv()
		if err != nil {
			return nil, err
		}
		if event == nil {
			continue
		}
		switch event.Type {
		case agent.StreamEventToken:
			if event.Token != "" {
				return &ai.Response{Reply: event.Token}, nil
			}
		case agent.StreamEventDone:
			return nil, io.EOF
		}
	}
}

func (s agentStreamAdapter) Close() error { return s.stream.Close() }

func main() {
	provider := flag.String("provider", "mock", "LLM provider; mock is deterministic and requires no API key")
	flag.Parse()
	if *provider == "mock" {
		ai.Register("mock", newMock)
	}

	fmt.Printf("\n\033[1mA2A streaming conformance (provider: %s)\033[0m\n", *provider)
	reg := registry.NewMemoryRegistry()
	st := store.NewMemoryStore()
	var sawTool, sawRunInfo bool
	ag := agent.New(append([]agent.Option{
		agent.Name("a2a-streaming"),
		agent.Provider(*provider),
		agent.Prompt("Use stream_echo exactly once with value a2a-stream, then answer with the tool result."),
		agent.WithRegistry(reg),
		agent.WithStore(st),
		agent.WithMemory(agent.NewInMemory(8)),
		agent.ModelCallTimeout(45 * time.Second),
		agent.WithTool("stream_echo", "Echo the A2A stream marker.", map[string]any{
			"value": map[string]any{"type": "string", "description": "value to echo"},
		}, func(ctx context.Context, input map[string]any) (string, error) {
			sawTool = true
			info, ok := ai.RunInfoFrom(ctx)
			if !ok || info.RunID == "" || info.Agent != "a2a-streaming" {
				return "", fmt.Errorf("unexpected run info: %+v", info)
			}
			sawRunInfo = true
			if input["value"] != "a2a-stream" {
				return "", fmt.Errorf("unexpected value %v", input["value"])
			}
			return `{"marker":"a2a-stream-ok"}`, nil
		}),
	}, harnessutil.AgentOptions(*provider)...)...)

	handler := a2a.NewAgentStreamHandler(
		a2a.Card("a2a-streaming", "http://example.invalid/a2a-streaming", "", nil),
		func(ctx context.Context, text string) (string, error) {
			resp, err := ag.Ask(ctx, text)
			if err != nil {
				return "", err
			}
			return resp.Reply, nil
		},
		func(ctx context.Context, text string) (ai.Stream, error) {
			stream, err := agent.StreamAsk(ctx, ag, text)
			if err != nil {
				return nil, err
			}
			return agentStreamAdapter{stream: stream}, nil
		},
	)

	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"message/stream","params":{"message":{"role":"user","parts":[{"kind":"text","text":"Run the A2A streaming conformance check."}],"kind":"message"}}}`)
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
	if summary.WorkingEvents == 0 || summary.State != "completed" || !strings.Contains(summary.FinalText, "a2a-stream-ok") {
		fmt.Fprintf(os.Stderr, "unexpected stream summary: %+v\npayload:\n%s", summary, summary.Payload)
		os.Exit(1)
	}
	if !sawTool || !sawRunInfo {
		fmt.Fprintf(os.Stderr, "tool=%v runInfo=%v\n", sawTool, sawRunInfo)
		os.Exit(1)
	}
	fmt.Println("\n\033[32m✓ A2A message/stream emitted incremental task updates and preserved tool/run metadata\033[0m")
}

type streamSummary struct {
	Payload       string
	State         string
	FinalText     string
	WorkingEvents int
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
			Error any `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &envelope); err != nil {
			return fmt.Errorf("SSE data event is not JSON: %s", data)
		}
		if envelope.Error != nil {
			return fmt.Errorf("SSE data event has error: %s", data)
		}
		seen = true
		summary.Payload += data + "\n"
		if envelope.Result.Status.State == "working" {
			summary.WorkingEvents++
		}
		if envelope.Result.Status.State != "" {
			summary.State = envelope.Result.Status.State
		}
		for _, artifact := range envelope.Result.Artifacts {
			for _, part := range artifact.Parts {
				summary.FinalText = part.Text
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
