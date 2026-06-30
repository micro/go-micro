package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/gateway/a2a"
)

func TestA2AStreamUsesAgentChatPathWithTools(t *testing.T) {
	var sawTool bool
	fakeGen = func(ctx context.Context, opts ai.Options, req *ai.Request) (*ai.Response, error) {
		if opts.ToolHandler == nil {
			t.Fatal("model was not wired with agent tool handler")
		}
		result := opts.ToolHandler(ctx, ai.ToolCall{
			ID:    "call-1",
			Name:  "echo",
			Input: map[string]any{"value": "a2a-stream"},
		})
		if !strings.Contains(result.Content, "a2a-stream-ok") {
			t.Fatalf("tool result = %q, want marker", result.Content)
		}
		return &ai.Response{Answer: "streamed " + result.Content}, nil
	}
	defer func() { fakeGen = nil }()

	a := newTestAgent(Name("stream-agent"), WithTool("echo", "echo text", nil, func(ctx context.Context, input map[string]any) (string, error) {
		sawTool = true
		if info, ok := ai.RunInfoFrom(ctx); !ok || info.RunID == "" || info.Agent != "stream-agent" {
			t.Fatalf("RunInfo = %+v ok=%v, want stream-agent run", info, ok)
		}
		if input["value"] != "a2a-stream" {
			t.Fatalf("tool input = %+v, want a2a-stream", input)
		}
		return "a2a-stream-ok", nil
	}))
	h := a2a.NewAgentStreamHandler(
		a2a.Card("stream-agent", "http://example.invalid/stream-agent", "", nil),
		func(ctx context.Context, text string) (string, error) {
			resp, err := a.Ask(ctx, text)
			if err != nil {
				return "", err
			}
			return resp.Reply, nil
		},
		a.streamAskAI,
	)

	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"message/stream","params":{"message":{"role":"user","parts":[{"kind":"text","text":"run stream tool"}],"kind":"message"}}}`)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if !sawTool {
		t.Fatal("A2A stream did not execute the agent tool path")
	}
	if ct := rr.Result().Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content-type = %q, want text/event-stream", ct)
	}
	if !strings.Contains(rr.Body.String(), "a2a-stream-ok") {
		t.Fatalf("stream body missing tool marker: %s", rr.Body.String())
	}

	var final struct {
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
	for _, line := range strings.Split(strings.TrimSpace(rr.Body.String()), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "data: "))
		if line == "" {
			continue
		}
		if err := json.Unmarshal([]byte(line), &final); err != nil {
			t.Fatalf("decode event %q: %v", line, err)
		}
	}
	if final.Error != nil {
		t.Fatalf("final event error: %+v", final.Error)
	}
	if final.Result.Status.State != "completed" {
		t.Fatalf("final state = %q, want completed", final.Result.Status.State)
	}
	if len(final.Result.Artifacts) != 1 || len(final.Result.Artifacts[0].Parts) != 1 || !strings.Contains(final.Result.Artifacts[0].Parts[0].Text, "a2a-stream-ok") {
		t.Fatalf("final artifacts = %+v, want tool marker", final.Result.Artifacts)
	}
}
