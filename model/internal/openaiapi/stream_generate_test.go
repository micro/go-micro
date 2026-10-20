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

// Gemini's OpenAI-compatible endpoint streams each parallel call whole, in its own chunk, without tool_calls[].index.
func TestGenerateKeepsUnindexedParallelToolCallsApart(t *testing.T) {
	requests := 0
	var seen []model.ToolCall
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "text/event-stream")
		if requests == 1 {
			fmt.Fprintln(w, `data: {"choices":[{"index":0,"delta":{"tool_calls":[{"id":"call_paris","function":{"name":"weather","arguments":"{\"city\":\"Paris\"}"}}]}}]}`)
			fmt.Fprintln(w, `data: {"choices":[{"index":0,"delta":{"tool_calls":[{"id":"call_london","function":{"name":"weather","arguments":"{\"city\":\"London\"}"}}]}}]}`)
			fmt.Fprintln(w, `data: {"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`)
		} else {
			fmt.Fprintln(w, `data: {"choices":[{"index":0,"delta":{"content":"done"},"finish_reason":"stop"}]}`)
		}
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer server.Close()
	opts := model.Options{BaseURL: server.URL, ToolHandler: func(_ context.Context, call model.ToolCall) model.ToolResult {
		seen = append(seen, call)
		return model.ToolResult{Content: "ok"}
	}}
	resp, err := Generate(context.Background(), opts, &model.Request{Tools: []model.Tool{{Name: "weather"}}}, nil, model.WithTokenHandler(func(string) {}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Answer != "done" || len(seen) != 2 {
		t.Fatalf("answer=%q calls=%+v", resp.Answer, seen)
	}
	if seen[0].ID != "call_paris" || seen[0].Input["city"] != "Paris" || seen[1].ID != "call_london" || seen[1].Input["city"] != "London" {
		t.Fatalf("calls were merged or reordered: %+v", seen)
	}
}

func TestGenerateContinuesUnindexedToolCallFragments(t *testing.T) {
	requests := 0
	var seen []model.ToolCall
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "text/event-stream")
		if requests == 1 {
			fmt.Fprintln(w, `data: {"choices":[{"index":0,"delta":{"tool_calls":[{"id":"call1","function":{"name":"echo","arguments":"{\"text\":"}}]}}]}`)
			fmt.Fprintln(w, `data: {"choices":[{"index":0,"delta":{"tool_calls":[{"function":{"arguments":"\"hello\"}"}}]}}]}`)
			fmt.Fprintln(w, `data: {"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`)
		} else {
			fmt.Fprintln(w, `data: {"choices":[{"index":0,"delta":{"content":"done"},"finish_reason":"stop"}]}`)
		}
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer server.Close()
	opts := model.Options{BaseURL: server.URL, ToolHandler: func(_ context.Context, call model.ToolCall) model.ToolResult {
		seen = append(seen, call)
		return model.ToolResult{Content: "ok"}
	}}
	if _, err := Generate(context.Background(), opts, &model.Request{Tools: []model.Tool{{Name: "echo"}}}, nil, model.WithTokenHandler(func(string) {})); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 || seen[0].ID != "call1" || seen[0].Input["text"] != "hello" {
		t.Fatalf("fragments were not joined into one call: %+v", seen)
	}
}
