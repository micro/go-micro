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

	// The spec-shaped stream carries the answer as append artifact-update
	// deltas and closes with a completed status-update (final:true).
	var (
		text       strings.Builder
		finalState string
		sawFinal   bool
	)
	for _, line := range strings.Split(strings.TrimSpace(rr.Body.String()), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "data: "))
		if line == "" {
			continue
		}
		var ev struct {
			Result json.RawMessage `json:"result"`
			Error  any             `json:"error"`
		}
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("decode event %q: %v", line, err)
		}
		if ev.Error != nil {
			t.Fatalf("event carried an error field: %+v", ev.Error)
		}
		var kind struct {
			Kind string `json:"kind"`
		}
		_ = json.Unmarshal(ev.Result, &kind)
		switch kind.Kind {
		case "artifact-update":
			var au struct {
				Artifact struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"artifact"`
			}
			_ = json.Unmarshal(ev.Result, &au)
			for _, p := range au.Artifact.Parts {
				text.WriteString(p.Text)
			}
		case "status-update":
			var su struct {
				Status struct {
					State string `json:"state"`
				} `json:"status"`
				Final bool `json:"final"`
			}
			_ = json.Unmarshal(ev.Result, &su)
			if su.Final {
				sawFinal = true
				finalState = su.Status.State
			}
		default: // opening "task" snapshot
			var task struct {
				Artifacts []struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"artifacts"`
			}
			_ = json.Unmarshal(ev.Result, &task)
			for _, a := range task.Artifacts {
				for _, p := range a.Parts {
					if p.Text != "" {
						text.WriteString(p.Text)
					}
				}
			}
		}
	}
	if !sawFinal || finalState != "completed" {
		t.Fatalf("want a completed final:true status-update; sawFinal=%v state=%q", sawFinal, finalState)
	}
	if !strings.Contains(text.String(), "a2a-stream-ok") {
		t.Fatalf("reassembled stream text missing tool marker: %q", text.String())
	}
}
