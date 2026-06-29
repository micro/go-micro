package atlascloud

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-micro.dev/v6/ai"
)

func TestProvider_String(t *testing.T) {
	p := NewProvider()
	if p.String() != "atlascloud" {
		t.Errorf("Expected provider name 'atlascloud', got '%s'", p.String())
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
	if opts.Model != "deepseek-ai/DeepSeek-V3-0324" {
		t.Errorf("Expected default model 'deepseek-ai/DeepSeek-V3-0324', got '%s'", opts.Model)
	}
	if opts.BaseURL != "https://api.atlascloud.ai" {
		t.Errorf("Expected default base URL 'https://api.atlascloud.ai', got '%s'", opts.BaseURL)
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
	var sawStream, sawIncludeUsage bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %s, want /v1/chat/completions", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		sawStream, _ = body["stream"].(bool)
		if so, ok := body["stream_options"].(map[string]any); ok {
			sawIncludeUsage, _ = so["include_usage"].(bool)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hel\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[],\"usage\":{\"prompt_tokens\":7,\"completion_tokens\":2,\"total_tokens\":9}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer ts.Close()

	p := NewProvider(ai.WithAPIKey("test-key"), ai.WithBaseURL(ts.URL))
	stream, err := p.Stream(context.Background(), &ai.Request{Prompt: "Hello"})
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}
	defer stream.Close()
	if !sawStream {
		t.Fatal("stream request did not set stream=true")
	}
	if !sawIncludeUsage {
		t.Fatal("stream request did not set stream_options.include_usage=true")
	}

	first, err := stream.Recv()
	if err != nil || first.Reply != "hel" {
		t.Fatalf("first chunk = %#v, %v; want hel", first, err)
	}
	second, err := stream.Recv()
	if err != nil || second.Reply != "lo" {
		t.Fatalf("second chunk = %#v, %v; want lo", second, err)
	}
	usage, err := stream.Recv()
	if err != nil {
		t.Fatalf("usage chunk error: %v", err)
	}
	if usage.Usage.TotalTokens != 9 || usage.Usage.InputTokens != 7 || usage.Usage.OutputTokens != 2 {
		t.Fatalf("usage = %#v; want input=7 output=2 total=9", usage.Usage)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("final error = %v, want EOF", err)
	}
}

func TestProvider_Registration(t *testing.T) {
	m := ai.New("atlascloud", ai.WithAPIKey("test"))
	if m == nil {
		t.Fatal("ai.New('atlascloud') returned nil — provider not registered")
	}
	if m.String() != "atlascloud" {
		t.Errorf("Expected 'atlascloud', got '%s'", m.String())
	}
}

func TestProvider_ImageRegistration(t *testing.T) {
	ig := ai.NewImage("atlascloud", ai.WithAPIKey("test"))
	if ig == nil {
		t.Fatal("ai.NewImage('atlascloud') returned nil — image provider not registered")
	}
	if ig.String() != "atlascloud" {
		t.Errorf("Expected 'atlascloud', got '%s'", ig.String())
	}
}

func TestProvider_GenerateImage_NoAPIKey(t *testing.T) {
	p := NewProvider()
	_, err := p.GenerateImage(context.Background(), &ai.ImageRequest{Prompt: "a cat"})
	if err == nil {
		t.Error("Expected error when API key is missing, got nil")
	}
}

func TestProvider_ImplementsImageModel(t *testing.T) {
	var _ ai.ImageModel = (*Provider)(nil)
}

func TestProvider_VideoRegistration(t *testing.T) {
	vg := ai.NewVideo("atlascloud", ai.WithAPIKey("test"))
	if vg == nil {
		t.Fatal("ai.NewVideo('atlascloud') returned nil — video provider not registered")
	}
	if vg.String() != "atlascloud" {
		t.Errorf("Expected 'atlascloud', got '%s'", vg.String())
	}
}

func TestProvider_GenerateVideo_NoAPIKey(t *testing.T) {
	p := NewProvider()
	_, err := p.GenerateVideo(context.Background(), &ai.VideoRequest{Prompt: "a cat"})
	if err == nil {
		t.Error("Expected error when API key is missing, got nil")
	}
}

func TestProvider_ImplementsVideoModel(t *testing.T) {
	var _ ai.VideoModel = (*Provider)(nil)
}
