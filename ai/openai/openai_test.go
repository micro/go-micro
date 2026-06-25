package openai

import (
	"context"
	"errors"
	"testing"

	"go-micro.dev/v6/ai"
)

func TestProvider_String(t *testing.T) {
	p := NewProvider()
	if p.String() != "openai" {
		t.Errorf("Expected provider name 'openai', got '%s'", p.String())
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
	if opts.Model != "gpt-4o" {
		t.Errorf("Expected default model 'gpt-4o', got '%s'", opts.Model)
	}
	if opts.BaseURL != "https://api.openai.com" {
		t.Errorf("Expected default base URL 'https://api.openai.com', got '%s'", opts.BaseURL)
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

func TestProvider_ImageRegistration(t *testing.T) {
	ig := ai.NewImage("openai", ai.WithAPIKey("test"))
	if ig == nil {
		t.Fatal("ai.NewImage('openai') returned nil — image provider not registered")
	}
	if ig.String() != "openai" {
		t.Errorf("Expected 'openai', got '%s'", ig.String())
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
