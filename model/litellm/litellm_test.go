package litellm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"go-micro.dev/v6/model"
)

func TestProvider_Defaults(t *testing.T) {
	p := NewProvider()
	if p.String() != "litellm" {
		t.Errorf("String() = %q", p.String())
	}
	opts := p.Options()
	if opts.Model != defaultModel || opts.BaseURL != defaultBaseURL || opts.APIKey != "" {
		t.Errorf("defaults = %+v", opts)
	}
	caps := model.ProviderCapabilities("litellm")
	if !caps.Model || !caps.Image || !caps.Stream || !caps.ToolStream {
		t.Errorf("capabilities = %+v", caps)
	}
	if model.APIKeyEnv("litellm") != "LITELLM_API_KEY" {
		t.Errorf("APIKeyEnv = %q", model.APIKeyEnv("litellm"))
	}
	if got := model.AutoDetectProvider("https://litellm.internal.example.com"); got != "litellm" {
		t.Errorf("AutoDetectProvider = %q", got)
	}
}

func TestGenerateWithoutKeySendsNoAuthHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("unexpected Authorization header %q", auth)
		}
		fmt.Fprint(w, `{"choices":[{"finish_reason":"stop","message":{"content":"pong"}}],"usage":{"prompt_tokens":3,"completion_tokens":1,"total_tokens":4}}`)
	}))
	defer server.Close()

	p := NewProvider(model.WithBaseURL(server.URL), model.WithModel("claude-sonnet"))
	resp, err := p.Generate(context.Background(), &model.Request{Prompt: "ping"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Reply != "pong" || resp.Usage.TotalTokens != 4 {
		t.Fatalf("response = %+v", resp)
	}
}

func TestToolRoundTripWithVirtualKey(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-virtual" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		calls++
		if calls == 1 {
			if body["model"] != "bedrock-claude" || body["tools"] == nil {
				t.Errorf("first request = %v", body)
			}
			fmt.Fprint(w, `{"choices":[{"finish_reason":"tool_calls","message":{"content":null,"tool_calls":[{"id":"call1","type":"function","function":{"name":"lookup","arguments":"{\"city\":\"Oslo\"}"}}]}}]}`)
			return
		}
		messages := body["messages"].([]any)
		result := messages[len(messages)-1].(map[string]any)
		if result["tool_call_id"] != "call1" || result["content"] != "4C" {
			t.Errorf("tool result = %v", result)
		}
		fmt.Fprint(w, `{"choices":[{"finish_reason":"stop","message":{"content":"It is 4C in Oslo."}}]}`)
	}))
	defer server.Close()

	executed := 0
	p := NewProvider(
		model.WithBaseURL(server.URL+"/v1"),
		model.WithAPIKey("sk-virtual"),
		model.WithModel("bedrock-claude"),
		model.WithToolHandler(func(_ context.Context, call model.ToolCall) model.ToolResult {
			executed++
			if call.Name != "lookup" || call.Input["city"] != "Oslo" {
				t.Errorf("tool call = %+v", call)
			}
			return model.ToolResult{Content: "4C"}
		}),
	)
	resp, err := p.Generate(context.Background(), &model.Request{Prompt: "weather?", Tools: []model.Tool{{Name: "lookup"}}})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || executed != 1 || resp.Answer != "It is 4C in Oslo." {
		t.Fatalf("calls=%d executed=%d response=%+v", calls, executed, resp)
	}
}

func TestModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" || r.Header.Get("Authorization") != "Bearer sk-virtual" {
			t.Errorf("request = %s %q", r.URL.Path, r.Header.Get("Authorization"))
		}
		fmt.Fprint(w, `{"data":[{"id":"gpt-4o-mini"},{"id":"bedrock-claude"},{"id":"gpt-4o-mini"}]}`)
	}))
	defer server.Close()

	p := NewProvider(model.WithBaseURL(server.URL), model.WithAPIKey("sk-virtual"))
	models, err := model.ListModels(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(models, []string{"bedrock-claude", "gpt-4o-mini"}) {
		t.Fatalf("models = %v", models)
	}
}

func TestGenerateImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/generations" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		want := map[string]any{"model": "imagen", "prompt": "a lighthouse", "drop_params": true, "size": "1024x1024", "n": float64(1)}
		if !reflect.DeepEqual(body, want) {
			t.Errorf("body = %v", body)
		}
		fmt.Fprint(w, `{"created":0,"data":[{"b64_json":"aW1n"}]}`)
	}))
	defer server.Close()

	ig := model.NewImage("litellm", model.WithBaseURL(server.URL))
	resp, err := ig.GenerateImage(context.Background(), &model.ImageRequest{Prompt: "a lighthouse", Model: "imagen", Size: "1024x1024", N: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Images) != 1 || resp.Images[0].Base64 != "aW1n" {
		t.Fatalf("images = %+v", resp.Images)
	}
	if _, err := ig.GenerateImage(context.Background(), &model.ImageRequest{Prompt: " "}); err == nil {
		t.Fatal("empty prompt accepted")
	}
}

func TestProxyErrorIsReturned(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"message":"Authentication Error, Invalid proxy server token passed.","type":"auth_error","code":"401"}}`)
	}))
	defer server.Close()

	p := NewProvider(model.WithBaseURL(server.URL), model.WithAPIKey("sk-wrong"))
	_, err := p.Generate(context.Background(), &model.Request{Prompt: "hi"})
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("err = %v", err)
	}
}

func TestStreamWithoutKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("unexpected Authorization header %q", auth)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"1, 2\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\", 3\"},\"finish_reason\":\"stop\"}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	p := NewProvider(model.WithBaseURL(server.URL))
	stream, err := p.Stream(context.Background(), &model.Request{Prompt: "count"})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	var text string
	for {
		chunk, err := stream.Recv()
		if err != nil {
			break
		}
		text += chunk.Reply
	}
	if text != "1, 2, 3" {
		t.Fatalf("streamed %q", text)
	}
}
