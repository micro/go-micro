package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-micro.dev/v6/ai"
)

func TestProvider_GenerateReasoningOptionsAndStopReason(t *testing.T) {
	var body map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"done"}],"stop_reason":"max_tokens"}`))
	}))
	defer ts.Close()

	p := NewProvider(ai.WithAPIKey("test-key"), ai.WithBaseURL(ts.URL),
		ai.WithThinking(ai.ThinkingAdaptive), ai.WithEffort("low"))
	resp, err := p.Generate(context.Background(), &ai.Request{Prompt: "Hello"})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if got := body["thinking"].(map[string]any)["type"]; got != "adaptive" {
		t.Fatalf("thinking type = %v, want adaptive", got)
	}
	if got := body["output_config"].(map[string]any)["effort"]; got != "low" {
		t.Fatalf("effort = %v, want low", got)
	}
	if resp.StopReason != "max_tokens" {
		t.Fatalf("stop reason = %q, want max_tokens", resp.StopReason)
	}
}

func TestProvider_String(t *testing.T) {
	p := NewProvider()
	if p.String() != "anthropic" {
		t.Errorf("Expected provider name 'anthropic', got '%s'", p.String())
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

func TestProvider_Options(t *testing.T) {
	p := NewProvider(
		ai.WithModel("custom-model"),
		ai.WithAPIKey("my-key"),
	)

	opts := p.Options()
	if opts.Model != "custom-model" {
		t.Errorf("Expected model 'custom-model', got '%s'", opts.Model)
	}
	if opts.APIKey != "my-key" {
		t.Errorf("Expected API key 'my-key', got '%s'", opts.APIKey)
	}
}

func TestProvider_Defaults(t *testing.T) {
	p := NewProvider()

	opts := p.Options()
	if opts.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Expected default model 'claude-sonnet-4-20250514', got '%s'", opts.Model)
	}
	if opts.BaseURL != "https://api.anthropic.com" {
		t.Errorf("Expected default base URL 'https://api.anthropic.com', got '%s'", opts.BaseURL)
	}
}

func TestProvider_Generate_NoAPIKey(t *testing.T) {
	p := NewProvider()

	req := &ai.Request{
		Prompt:       "Hello",
		SystemPrompt: "You are helpful",
	}

	_, err := p.Generate(context.Background(), req)
	if err == nil {
		t.Error("Expected error when API key is missing, got nil")
	}
}

func TestProvider_Stream(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("path = %q, want /v1/messages", r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != "text/event-stream" {
			t.Fatalf("Accept = %q, want text/event-stream", got)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Fatalf("x-api-key = %q, want test-key", got)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"stream":true`) {
			t.Fatalf("request body %s does not enable streaming", string(body))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("event: message_start\n"))
		_, _ = w.Write([]byte(`data: {"type":"message_start","message":{"usage":{"input_tokens":2}}}` + "\n\n"))
		_, _ = w.Write([]byte("event: content_block_delta\n"))
		_, _ = w.Write([]byte(`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"hel"}}` + "\n\n"))
		_, _ = w.Write([]byte("event: content_block_delta\n"))
		_, _ = w.Write([]byte(`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"lo"}}` + "\n\n"))
		_, _ = w.Write([]byte("event: message_delta\n"))
		_, _ = w.Write([]byte(`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":3}}` + "\n\n"))
		_, _ = w.Write([]byte("event: message_stop\n"))
		_, _ = w.Write([]byte(`data: {"type":"message_stop"}` + "\n\n"))
	}))
	defer ts.Close()

	p := NewProvider(ai.WithAPIKey("test-key"), ai.WithBaseURL(ts.URL))

	req := &ai.Request{
		Prompt: "Hello",
	}

	stream, err := p.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	defer stream.Close()

	var reply strings.Builder
	var usage ai.Usage
	var stopReason string
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Recv failed: %v", err)
		}
		reply.WriteString(chunk.Reply)
		if chunk.Usage.TotalTokens > 0 {
			usage = chunk.Usage
		}
		if chunk.StopReason != "" {
			stopReason = chunk.StopReason
		}
	}
	if got := reply.String(); got != "hello" {
		t.Fatalf("reply = %q, want hello", got)
	}
	if usage.TotalTokens != 3 {
		t.Fatalf("usage = %+v, want total 3", usage)
	}
	if stopReason != "end_turn" {
		t.Fatalf("stop reason = %q, want end_turn", stopReason)
	}
}
