package gemini

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

func TestProvider_String(t *testing.T) {
	p := NewProvider()
	if p.String() != "gemini" {
		t.Errorf("Expected provider name 'gemini', got '%s'", p.String())
	}
}

func TestProvider_Init(t *testing.T) {
	p := NewProvider()

	err := p.Init(
		ai.WithModel("gemini-2.0-flash"),
		ai.WithAPIKey("test-key"),
		ai.WithBaseURL("https://test.com"),
	)

	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	opts := p.Options()
	if opts.Model != "gemini-2.0-flash" {
		t.Errorf("Expected model 'gemini-2.0-flash', got '%s'", opts.Model)
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
	if opts.Model != "gemini-2.5-flash" {
		t.Errorf("Expected default model 'gemini-2.5-flash', got '%s'", opts.Model)
	}
	if opts.BaseURL != "https://generativelanguage.googleapis.com" {
		t.Errorf("Expected default base URL 'https://generativelanguage.googleapis.com', got '%s'", opts.BaseURL)
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
	var sawRequest bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawRequest = true
		if r.URL.Path != "/v1beta/models/gemini-2.5-flash:streamGenerateContent" {
			t.Fatalf("path = %s, want streamGenerateContent", r.URL.Path)
		}
		if r.URL.Query().Get("alt") != "sse" {
			t.Fatalf("alt = %q, want sse", r.URL.Query().Get("alt"))
		}
		if got := r.Header.Get("Accept"); got != "text/event-stream" {
			t.Fatalf("Accept = %q, want text/event-stream", got)
		}
		if got := r.Header.Get("x-goog-api-key"); got != "test-key" {
			t.Fatalf("x-goog-api-key = %q, want test-key", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		contents, ok := body["contents"].([]any)
		if !ok || len(contents) != 3 {
			t.Fatalf("contents = %#v, want history + prompt", body["contents"])
		}
		second := contents[1].(map[string]any)
		if second["role"] != "model" {
			t.Fatalf("assistant history role = %#v, want model", second["role"])
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"hel\"}]}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"lo\"}]}}],\"usageMetadata\":{\"promptTokenCount\":3,\"candidatesTokenCount\":2,\"totalTokenCount\":5}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer ts.Close()

	p := NewProvider(ai.WithAPIKey("test-key"), ai.WithBaseURL(ts.URL))
	stream, err := p.Stream(context.Background(), &ai.Request{
		Messages: []ai.Message{
			{Role: "user", Content: "previous question"},
			{Role: "assistant", Content: "previous answer"},
		},
		Prompt: "Hello",
	})
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}
	defer stream.Close()
	if !sawRequest {
		t.Fatal("server did not receive stream request")
	}

	first, err := stream.Recv()
	if err != nil || first.Reply != "hel" {
		t.Fatalf("first chunk = %#v, %v; want hel", first, err)
	}
	second, err := stream.Recv()
	if err != nil || second.Reply != "lo" {
		t.Fatalf("second chunk = %#v, %v; want lo", second, err)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("final error = %v, want EOF", err)
	}
}

func TestProvider_StreamPropagatesMalformedChunk(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {bad json}\n\n"))
	}))
	defer ts.Close()

	p := NewProvider(ai.WithAPIKey("test-key"), ai.WithBaseURL(ts.URL))
	stream, err := p.Stream(context.Background(), &ai.Request{Prompt: "Hello"})
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}
	defer stream.Close()

	if _, err := stream.Recv(); err == nil {
		t.Fatal("Recv returned nil error for malformed chunk")
	}
}

func TestProvider_StreamPropagatesProviderError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "quota exhausted", http.StatusTooManyRequests)
	}))
	defer ts.Close()

	p := NewProvider(ai.WithAPIKey("test-key"), ai.WithBaseURL(ts.URL))
	stream, err := p.Stream(context.Background(), &ai.Request{Prompt: "Hello"})
	if err == nil {
		_ = stream.Close()
		t.Fatal("Stream returned nil error for provider failure")
	}
	if !strings.Contains(err.Error(), "429") || !strings.Contains(err.Error(), "quota exhausted") {
		t.Fatalf("Stream error = %v, want provider status and body", err)
	}
	if strings.Contains(err.Error(), "test-key") {
		t.Fatal("stream error leaked API key")
	}
}

func TestProvider_Registration(t *testing.T) {
	m := ai.New("gemini", ai.WithAPIKey("test"))
	if m == nil {
		t.Fatal("ai.New('gemini') returned nil — provider not registered")
	}
	if m.String() != "gemini" {
		t.Errorf("Expected 'gemini', got '%s'", m.String())
	}
}
