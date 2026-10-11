package agent

import (
	"context"
	"strings"
	"testing"
)

func TestProviderEnvironment(t *testing.T) {
	for _, key := range []string{"MICRO_AI_PROVIDER", "MICRO_AI_MODEL", "MICRO_AI_BASE_URL", "MICRO_AI_API_KEY", "OPENAI_API_KEY", "MISTRAL_API_KEY"} {
		t.Setenv(key, "")
	}
	if got := New().Options().Provider; got != "openai" {
		t.Fatalf("default provider = %q", got)
	}
	t.Setenv("MICRO_AI_PROVIDER", "mistral")
	t.Setenv("MICRO_AI_MODEL", "custom-model")
	t.Setenv("MISTRAL_API_KEY", "mistral-test-key")
	t.Setenv("OPENAI_API_KEY", "openai-test-key")
	o := New().Options()
	if o.Provider != "mistral" || o.Model != "custom-model" || o.APIKey != "mistral-test-key" {
		t.Fatal("agent did not use provider environment")
	}
	o = New(Provider("openai"), Model("explicit-model")).Options()
	if o.Provider != "openai" || o.Model != "explicit-model" || o.APIKey != "openai-test-key" {
		t.Fatal("explicit provider did not select its own key")
	}
	t.Setenv("MICRO_AI_API_KEY", "generic-test-key")
	if New().Options().APIKey != "generic-test-key" {
		t.Fatal("generic key should have the same precedence as chat")
	}
	if New(APIKey("")).Options().APIKey != "" {
		t.Fatal("explicit empty key must disable environment authentication")
	}
	o = New(APIKey("explicit-key"), BaseURL("http://localhost:9999")).Options()
	if o.APIKey != "explicit-key" || o.BaseURL != "http://localhost:9999" {
		t.Fatal("explicit options must override environment")
	}
}

func TestUnknownProviderReturnsError(t *testing.T) {
	a := New(Provider("does-not-exist"), Services())
	_, err := a.Ask(context.Background(), "hello")
	if err == nil || !strings.Contains(err.Error(), "does-not-exist") {
		t.Fatalf("Ask error = %v", err)
	}
	_, err = a.Stream(context.Background(), "hello")
	if err == nil || !strings.Contains(err.Error(), "does-not-exist") {
		t.Fatalf("Stream error = %v", err)
	}
}
