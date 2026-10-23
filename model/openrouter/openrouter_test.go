package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-micro.dev/v6/model"
)

func TestToolRoundTrip(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected request: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if body["model"] != "test/model" || body["tools"] == nil {
			t.Errorf("missing model/tools: %v", body)
		}
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			fmt.Fprint(w, `{"choices":[{"finish_reason":"tool_calls","message":{"content":null,"reasoning_details":[{"type":"reasoning.encrypted","data":"opaque","index":0}],"tool_calls":[{"id":"call1","type":"function","function":{"name":"lookup","arguments":"{\"id\":1}"}}]}}]}`)
			return
		}
		messages := body["messages"].([]any)
		assistant := messages[len(messages)-2].(map[string]any)
		if assistant["reasoning_details"] == nil {
			t.Error("reasoning details lost")
		}
		result := messages[len(messages)-1].(map[string]any)
		if result["tool_call_id"] != "call1" || result["content"] != "found" {
			t.Errorf("bad tool result: %v", result)
		}
		fmt.Fprint(w, `{"choices":[{"finish_reason":"stop","message":{"content":"Done"}}],"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12}}`)
	}))
	defer server.Close()
	executed := 0
	provider := NewProvider(model.WithBaseURL(server.URL), model.WithAPIKey("test-key"), model.WithModel("test/model"), model.WithToolHandler(func(ctx context.Context, call model.ToolCall) model.ToolResult {
		executed++
		if call.Name != "lookup" || call.Input["id"] != float64(1) {
			t.Errorf("bad call: %v", call)
		}
		return model.ToolResult{Content: "found"}
	}))
	var output string
	response, err := provider.Generate(context.Background(), &model.Request{Prompt: "look up", Tools: []model.Tool{{Name: "lookup"}}}, model.WithTokenHandler(func(s string) { output += s }))
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || executed != 1 || response.Answer != "Done" || output != "Done" {
		t.Fatalf("calls=%d tools=%d response=%+v output=%q", calls, executed, response, output)
	}
}

func TestDefaultsAndCatalog(t *testing.T) {
	p := NewProvider()
	if p.String() != "openrouter" || p.Options().Model != "openrouter/auto" || p.Options().BaseURL != "https://openrouter.ai/api" {
		t.Fatalf("defaults: %+v", p.Options())
	}
	if model.New("openrouter") == nil || model.APIKeyEnv("openrouter") != "OPENROUTER_API_KEY" {
		t.Fatal("provider registration")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"data":[{"id":"vendor/model"}]}`)
	}))
	defer server.Close()
	if err := p.Init(model.WithBaseURL(server.URL)); err != nil {
		t.Fatal(err)
	}
	models, err := p.Models(context.Background())
	if err != nil || len(models) != 1 || models[0] != "vendor/model" {
		t.Fatalf("models=%v err=%v", models, err)
	}
}

func TestStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\ndata: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()
	stream, err := NewProvider(model.WithBaseURL(server.URL)).Stream(context.Background(), &model.Request{Prompt: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	chunk, err := stream.Recv()
	if err != nil || chunk.Reply != "Hello" {
		t.Fatalf("chunk=%v err=%v", chunk, err)
	}
}

func TestAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"message":"invalid key"}}`)
	}))
	defer server.Close()
	_, err := NewProvider(model.WithBaseURL(server.URL)).Generate(context.Background(), &model.Request{Prompt: "hello"})
	if err == nil {
		t.Fatal("expected authentication error")
	}
}

func TestToolCallWithoutArguments(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			fmt.Fprint(w, `{"choices":[{"finish_reason":"tool_calls","message":{"content":null,"tool_calls":[{"id":"call1","type":"function","function":{"name":"now","arguments":""}}]}}]}`)
			return
		}
		var request struct {
			Messages []struct {
				ToolCalls []struct {
					Function struct {
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		found := 0
		for _, message := range request.Messages {
			for _, call := range message.ToolCalls {
				found++
				if args := call.Function.Arguments; args != "{}" {
					t.Errorf("follow-up arguments = %q, want {}", args)
				}
			}
		}
		if found != 1 {
			t.Errorf("follow-up tool calls = %d, want 1", found)
		}

		fmt.Fprint(w, `{"choices":[{"finish_reason":"stop","message":{"content":"Done"}}]}`)
	}))
	defer server.Close()
	executed := 0
	provider := NewProvider(model.WithBaseURL(server.URL), model.WithAPIKey("test-key"), model.WithModel("test/model"), model.WithToolHandler(func(ctx context.Context, call model.ToolCall) model.ToolResult {
		executed++
		if len(call.Input) != 0 {
			t.Errorf("unexpected input: %v", call.Input)
		}
		return model.ToolResult{Content: "noon"}
	}))
	response, err := provider.Generate(context.Background(), &model.Request{Prompt: "time?", Tools: []model.Tool{{Name: "now"}}})
	if err != nil {
		t.Fatal(err)
	}
	if executed != 1 || response.Answer != "Done" {
		t.Fatalf("tools=%d response=%+v", executed, response)
	}
}
