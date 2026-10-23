package model_test

import (
	"go-micro.dev/v6/model"
	_ "go-micro.dev/v6/model/mistral"
	_ "go-micro.dev/v6/model/openai"
	"testing"
)

func TestProviderSelection(t *testing.T) {
	if m := model.New(""); m == nil || m.String() != "openai" {
		t.Fatal("empty provider must select OpenAI")
	}
	if model.New("misspelled-provider") != nil {
		t.Fatal("unknown provider must not silently select another provider")
	}
	if m := model.New("mistral"); m == nil || m.String() != "mistral" {
		t.Fatal("explicit provider must be preserved")
	}
}

func TestOpenRouterDetection(t *testing.T) {
	if got := model.AutoDetectProvider("https://litellm.example.com"); got != "litellm" {
		t.Errorf("AutoDetectProvider(litellm) = %q", got)
	}
	for _, endpoint := range []string{"https://openrouter.ai/api", "https://openrouter.ai/api/v1"} {
		if got := model.AutoDetectProvider(endpoint); got != "openrouter" {
			t.Fatalf("provider for %s = %s", endpoint, got)
		}
	}
}
