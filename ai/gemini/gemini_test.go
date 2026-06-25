package gemini

import (
	"context"
	"errors"
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

func TestProvider_Stream_NotImplemented(t *testing.T) {
	p := NewProvider()

	req := &ai.Request{
		Prompt: "Hello",
	}

	_, err := p.Stream(context.Background(), req)
	if !errors.Is(err, ai.ErrStreamingUnsupported) {
		t.Fatalf("Stream error = %v, want ErrStreamingUnsupported", err)
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
