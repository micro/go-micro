package flow

import "testing"

func TestProviderEnvironment(t *testing.T) {
	t.Setenv("MICRO_AI_PROVIDER", "mistral")
	t.Setenv("MICRO_AI_MODEL", "configured-model")
	t.Setenv("MICRO_AI_BASE_URL", "http://localhost:9999")
	t.Setenv("MICRO_AI_API_KEY", "")
	t.Setenv("MISTRAL_API_KEY", "mistral-test-key")
	t.Setenv("OPENAI_API_KEY", "openai-test-key")
	o := New("test").opts
	if o.Provider != "mistral" || o.Model != "configured-model" || o.APIKey != "mistral-test-key" || o.BaseURL != "http://localhost:9999" {
		t.Fatal("flow did not use configured environment")
	}
	if New("test", APIKey("")).opts.APIKey != "" {
		t.Fatal("explicit empty key must disable environment authentication")
	}
	o = New("test", Provider("openai"), Model("explicit"), BaseURL(""), APIKey("explicit-key")).opts
	if o.Provider != "openai" || o.Model != "explicit" || o.APIKey != "explicit-key" || o.BaseURL != "" {
		t.Fatal("explicit flow options did not override environment")
	}
}
