package ollama

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-micro.dev/v6/ai"
)

// ---------------------------------------------------------------------------
// Provider basics
// ---------------------------------------------------------------------------

func TestProvider_String(t *testing.T) {
	p := NewProvider()
	if p.String() != "ollama" {
		t.Errorf("Expected 'ollama', got '%s'", p.String())
	}
}

func TestProvider_Init(t *testing.T) {
	p := NewProvider()
	err := p.Init(
		ai.WithModel("test-model"),
		ai.WithAPIKey("test-key"),
		ai.WithBaseURL("https://test.com"),
	)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	opts := p.Options()
	if opts.Model != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", opts.Model)
	}
	if opts.APIKey != "test-key" {
		t.Errorf("Expected API key 'test-key', got '%s'", opts.APIKey)
	}
	if opts.BaseURL != "https://test.com" {
		t.Errorf("Expected base URL 'https://test.com', got '%s'", opts.BaseURL)
	}
}

func TestProvider_Defaults(t *testing.T) {
	p := NewProvider()
	opts := p.Options()
	if opts.Model != "llama3.2" {
		t.Errorf("Expected default model 'llama3.2', got '%s'", opts.Model)
	}
	if opts.BaseURL != "http://localhost:11434" {
		t.Errorf("Expected default base URL 'http://localhost:11434', got '%s'", opts.BaseURL)
	}
}

func TestProvider_IsCloud(t *testing.T) {
	local := NewProvider(ai.WithBaseURL("http://localhost:11434"))
	if local.isCloud() {
		t.Error("localhost should not be cloud")
	}
	cloud := NewProvider(ai.WithBaseURL("https://ollama.com/v1"))
	if !cloud.isCloud() {
		t.Error("ollama.com should be cloud")
	}
}

// ---------------------------------------------------------------------------
// Native mode (local Ollama: /api/chat)
// ---------------------------------------------------------------------------

func TestNative_Generate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("Expected /api/chat, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"model": "llama3.2",
			"message": {"role": "assistant", "content": "Hello from local Ollama!"},
			"done": true,
			"prompt_eval_count": 10,
			"eval_count": 5
		}`))
	}))
	defer srv.Close()

	p := NewProvider(ai.WithBaseURL(srv.URL), ai.WithModel("llama3.2"))
	resp, err := p.Generate(context.Background(), &ai.Request{
		Prompt:       "Hi",
		SystemPrompt: "You are helpful",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if resp.Reply != "Hello from local Ollama!" {
		t.Errorf("Expected 'Hello from local Ollama!', got '%s'", resp.Reply)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("Expected total tokens 15, got %d", resp.Usage.TotalTokens)
	}
}

func TestNative_GenerateWithToolCall(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			w.Write([]byte(`{
				"model": "llama3.2",
				"message": {
					"role": "assistant",
					"content": "",
					"tool_calls": [{"function": {"name": "get_weather", "arguments": "{\"city\":\"Seoul\"}"}}]
				},
				"done": true
			}`))
		} else {
			w.Write([]byte(`{
				"model": "llama3.2",
				"message": {"role": "assistant", "content": "The weather in Seoul is sunny."},
				"done": true
			}`))
		}
	}))
	defer srv.Close()

	handler := func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		if call.Name != "get_weather" {
			t.Errorf("Expected tool 'get_weather', got '%s'", call.Name)
		}
		return ai.ToolResult{ID: call.ID, Content: `{"temp": 22, "condition": "sunny"}`}
	}

	p := NewProvider(
		ai.WithBaseURL(srv.URL),
		ai.WithModel("llama3.2"),
		ai.WithToolHandler(handler),
	)
	resp, err := p.Generate(context.Background(), &ai.Request{
		Prompt: "What's the weather?",
		Tools: []ai.Tool{{
			Name:        "get_weather",
			Description: "Get weather",
			Properties:  map[string]any{"city": map[string]any{"type": "string"}},
		}},
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if len(resp.ToolCalls) == 0 {
		t.Error("Expected tool calls")
	}
	if resp.Answer != "The weather in Seoul is sunny." {
		t.Errorf("Expected final answer, got '%s'", resp.Answer)
	}
}

func TestNative_Stream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":{"role":"assistant","content":"Hello"},"done":false}` + "\n"))
		w.Write([]byte(`{"message":{"role":"assistant","content":" world"},"done":false}` + "\n"))
		w.Write([]byte(`{"message":{"role":"assistant","content":""},"done":true}` + "\n"))
	}))
	defer srv.Close()

	p := NewProvider(ai.WithBaseURL(srv.URL), ai.WithModel("llama3.2"))
	stream, err := p.Stream(context.Background(), &ai.Request{Prompt: "Hi"})
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	defer stream.Close()

	var chunks []string
	for {
		resp, err := stream.Recv()
		if err != nil {
			break
		}
		if resp.Reply != "" {
			chunks = append(chunks, resp.Reply)
		}
	}
	result := strings.Join(chunks, "")
	if result != "Hello world" {
		t.Errorf("Expected 'Hello world', got '%s'", result)
	}
}

// ---------------------------------------------------------------------------
// Cloud mode (Ollama Cloud: /v1/chat/completions)
// ---------------------------------------------------------------------------

func TestCloud_Generate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("Expected /v1/chat/completions, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15},
			"choices": [{"message": {"role": "assistant", "content": "Hello from Ollama Cloud!"}}]
		}`))
	}))
	defer srv.Close()

	p := NewProvider(ai.WithBaseURL(srv.URL), ai.WithModel("gemma4:31b-cloud"), ai.WithAPIKey("test-key"))
	p.cloudOverride = true
	resp, err := p.Generate(context.Background(), &ai.Request{
		Prompt:       "Hi",
		SystemPrompt: "You are helpful",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if resp.Reply != "Hello from Ollama Cloud!" {
		t.Errorf("Expected 'Hello from Ollama Cloud!', got '%s'", resp.Reply)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("Expected total tokens 15, got %d", resp.Usage.TotalTokens)
	}
}

func TestCloud_GenerateWithToolCall(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			w.Write([]byte(`{
				"choices": [{"message": {
					"role": "assistant",
					"content": "",
					"tool_calls": [{"id": "call_1", "function": {"name": "search", "arguments": "{\"query\":\"go interfaces\"}"}}]
				}}]
			}`))
		} else {
			w.Write([]byte(`{
				"choices": [{"message": {"role": "assistant", "content": "Go interfaces are implicit."}}]
			}`))
		}
	}))
	defer srv.Close()

	handler := func(ctx context.Context, call ai.ToolCall) ai.ToolResult {
		return ai.ToolResult{ID: call.ID, Content: `{"results": ["Go interfaces are implicit"]}`}
	}

	p := NewProvider(
		ai.WithBaseURL(srv.URL),
		ai.WithModel("gemma4:31b-cloud"),
		ai.WithAPIKey("test-key"),
		ai.WithToolHandler(handler),
	)
	p.cloudOverride = true
	resp, err := p.Generate(context.Background(), &ai.Request{
		Prompt: "Search for Go interfaces",
		Tools: []ai.Tool{{
			Name:        "search",
			Description: "Search the knowledge base",
			Properties:  map[string]any{"query": map[string]any{"type": "string"}},
		}},
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if len(resp.ToolCalls) == 0 {
		t.Error("Expected tool calls")
	}
	if resp.Answer != "Go interfaces are implicit." {
		t.Errorf("Expected final answer, got '%s'", resp.Answer)
	}
}

func TestCloud_Stream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n"))
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" cloud\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()

	p := NewProvider(
		ai.WithBaseURL(srv.URL),
		ai.WithModel("gemma4:31b-cloud"),
		ai.WithAPIKey("test-key"),
	)
	p.cloudOverride = true
	stream, err := p.Stream(context.Background(), &ai.Request{Prompt: "Hi"})
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	defer stream.Close()

	var chunks []string
	for {
		resp, err := stream.Recv()
		if err != nil {
			break
		}
		if resp.Reply != "" {
			chunks = append(chunks, resp.Reply)
		}
	}
	result := strings.Join(chunks, "")
	if result != "Hello cloud" {
		t.Errorf("Expected 'Hello cloud', got '%s'", result)
	}
}

// ---------------------------------------------------------------------------
// Error handling
// ---------------------------------------------------------------------------

func TestProvider_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "model not found"}`))
	}))
	defer srv.Close()

	p := NewProvider(ai.WithBaseURL(srv.URL), ai.WithModel("nonexistent"))
	_, err := p.Generate(context.Background(), &ai.Request{Prompt: "Hi"})
	if err == nil {
		t.Error("Expected error on API failure")
	}
	if !strings.Contains(err.Error(), "API error") {
		t.Errorf("Expected 'API error' in message, got '%s'", err.Error())
	}
}
