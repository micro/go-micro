package openaiapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-micro.dev/v6/model"
)

func TestGenerateStreamsTextAndExecutesCompleteToolCalls(t *testing.T) {
	var tokens strings.Builder
	calls := 0
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if body["stream"] != true || body["tools"] == nil {
			t.Error("missing streaming or tool definitions")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if requests == 1 {
			fmt.Fprintln(w, `data: {"choices":[{"index":0,"delta":{"content":"Checking\n"}}]}`)
			fmt.Fprintln(w, `data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call1","function":{"name":"echo","arguments":"{\"text\":"}}]}}]}`)
			fmt.Fprintln(w, `data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"hello\"}"}}]},"finish_reason":"tool_calls"}]}`)
		} else {
			messages := body["messages"].([]any)
			last := messages[len(messages)-1].(map[string]any)
			if last["role"] != "tool" || last["content"] != "hello" {
				t.Errorf("lost tool result: %v", last)
			}
			fmt.Fprintln(w, `data: {"choices":[{"index":0,"delta":{"content":"Done\n\n  code"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`)
		}
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer server.Close()
	opts := model.Options{BaseURL: server.URL, ToolHandler: func(_ context.Context, call model.ToolCall) model.ToolResult {
		calls++
		if tokens.String() != "Checking\n" {
			t.Errorf("text was not delivered before the tool: %q", tokens.String())
		}
		return model.ToolResult{Content: call.Input["text"].(string)}
	}}
	resp, err := Generate(context.Background(), opts, &model.Request{Tools: []model.Tool{{Name: "echo"}}}, nil, model.WithTokenHandler(func(s string) { tokens.WriteString(s) }))
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || requests != 2 || resp.Answer != "Done\n\n  code" || tokens.String() != "Checking\nDone\n\n  code" {
		t.Fatalf("calls=%d requests=%d response=%+v tokens=%q", calls, requests, resp, tokens.String())
	}
}

func TestTruncatedToolStreamDoesNotExecute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"x","function":{"name":"write","arguments":"{\"path\":"}}]}}]}`)
	}))
	defer server.Close()
	_, err := Generate(context.Background(), model.Options{BaseURL: server.URL, ToolHandler: func(context.Context, model.ToolCall) model.ToolResult {
		t.Error("executed truncated tool call")
		return model.ToolResult{}
	}}, &model.Request{}, nil, model.WithTokenHandler(func(string) {}))
	if err == nil {
		t.Fatal("expected truncated stream error")
	}
}
