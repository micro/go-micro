package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-micro.dev/v6/ai"
)

// A multi-step task needs the provider to (a) loop while the model keeps
// calling tools and (b) keep offering the tools on every follow-up request.
// Without (a) a second step is impossible; without (b) the model writes the
// call it wanted as prose. This pins both, plus the loop bound.
func TestGenerateToolLoopKeepsOfferingTools(t *testing.T) {
	var requests []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		requests = append(requests, body)

		call := len(requests)
		w.Header().Set("Content-Type", "application/json")
		if call <= 2 {
			// Rounds 1 and 2: the model asks for a tool.
			fmt.Fprintf(w, `{"choices":[{"message":{"role":"assistant","content":"",
				"tool_calls":[{"id":"call-%d","type":"function","function":{"name":"step","arguments":"{\"n\":%d}"}}]}}]}`, call, call)
			return
		}
		// Round 3: done.
		fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"all steps done"}}]}`)
	}))
	defer srv.Close()

	toolRuns := 0
	p := &Provider{}
	if err := p.Init(
		ai.WithAPIKey("test"),
		ai.WithBaseURL(srv.URL),
		ai.WithModel("test-model"),
		ai.WithToolHandler(func(ctx context.Context, tc ai.ToolCall) ai.ToolResult {
			toolRuns++
			return ai.ToolResult{ID: tc.ID, Content: "done"}
		}),
	); err != nil {
		t.Fatalf("Init: %v", err)
	}

	resp, err := p.Generate(context.Background(), &ai.Request{
		Prompt: "do a two-step task",
		Tools:  []ai.Tool{{Name: "step", Description: "one step", Properties: map[string]any{}}},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if len(requests) != 3 {
		t.Fatalf("model calls = %d, want 3 (two tool rounds + final answer)", len(requests))
	}
	if toolRuns != 2 {
		t.Fatalf("tool executions = %d, want 2", toolRuns)
	}
	// Every follow-up must keep the tools on offer.
	for i, req := range requests[1:] {
		if _, ok := req["tools"]; !ok {
			t.Errorf("follow-up request %d omitted tools — the model is being asked to continue with its hands tied", i+1)
		}
	}
	if resp.Answer != "all steps done" {
		t.Errorf("Answer = %q, want the final reply", resp.Answer)
	}
	if len(resp.ToolCalls) != 2 {
		t.Errorf("recorded tool calls = %d, want 2", len(resp.ToolCalls))
	}
}

// A model that never stops asking for tools must be cut off at the bound
// rather than looping forever.
func TestGenerateToolLoopIsBounded(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"choices":[{"message":{"role":"assistant","content":"",
			"tool_calls":[{"id":"call-%d","type":"function","function":{"name":"step","arguments":"{}"}}]}}]}`, calls)
	}))
	defer srv.Close()

	p := &Provider{}
	if err := p.Init(
		ai.WithAPIKey("test"),
		ai.WithBaseURL(srv.URL),
		ai.WithModel("test-model"),
		ai.WithToolHandler(func(ctx context.Context, tc ai.ToolCall) ai.ToolResult {
			return ai.ToolResult{ID: tc.ID, Content: "done"}
		}),
	); err != nil {
		t.Fatalf("Init: %v", err)
	}

	if _, err := p.Generate(context.Background(), &ai.Request{
		Prompt: "never finish",
		Tools:  []ai.Tool{{Name: "step", Description: "one step", Properties: map[string]any{}}},
	}); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Initial call + at most maxToolRounds follow-ups.
	if calls > maxToolRounds+1 {
		t.Fatalf("model calls = %d, want at most %d — the loop must be bounded", calls, maxToolRounds+1)
	}
	if calls < maxToolRounds {
		t.Fatalf("model calls = %d — expected the loop to keep going while tool calls keep coming", calls)
	}
}
