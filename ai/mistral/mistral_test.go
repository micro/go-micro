package mistral

import (
	"context"
	"errors"
	"testing"

	"go-micro.dev/v6/ai"
)

func TestProvider_String(t *testing.T) {
	if NewProvider().String() != "mistral" {
		t.Errorf("got %q", NewProvider().String())
	}
}

func TestProvider_Defaults(t *testing.T) {
	opts := NewProvider().Options()
	if opts.Model != "mistral-large-latest" {
		t.Errorf("default model = %q", opts.Model)
	}
	if opts.BaseURL != "https://api.mistral.ai" {
		t.Errorf("default base URL = %q", opts.BaseURL)
	}
}

func TestProvider_Init(t *testing.T) {
	p := NewProvider()
	if err := p.Init(ai.WithModel("m"), ai.WithAPIKey("k")); err != nil {
		t.Fatal(err)
	}
	if p.Options().Model != "m" || p.Options().APIKey != "k" {
		t.Error("Init did not apply options")
	}
}

func TestProvider_Generate_NoAPIKey(t *testing.T) {
	if _, err := NewProvider().Generate(context.Background(), &ai.Request{Prompt: "hi"}); err == nil {
		t.Error("expected error without API key")
	}
}

func TestProvider_Stream_NotImplemented(t *testing.T) {
	if _, err := NewProvider().Stream(context.Background(), &ai.Request{Prompt: "hi"}); !errors.Is(err, ai.ErrStreamingUnsupported) {
		t.Fatalf("Stream error = %v, want ErrStreamingUnsupported", err)
	}
}

func TestProvider_Registration(t *testing.T) {
	m := ai.New("mistral", ai.WithAPIKey("test"))
	if m == nil {
		t.Fatal("provider not registered")
	}
	if m.String() != "mistral" {
		t.Errorf("got %q", m.String())
	}
}
