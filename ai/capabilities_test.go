package ai_test

import (
	"reflect"
	"testing"

	"go-micro.dev/v6/ai"
	_ "go-micro.dev/v6/ai/anthropic"
	_ "go-micro.dev/v6/ai/atlascloud"
	_ "go-micro.dev/v6/ai/gemini"
	_ "go-micro.dev/v6/ai/groq"
	_ "go-micro.dev/v6/ai/mistral"
	_ "go-micro.dev/v6/ai/openai"
	_ "go-micro.dev/v6/ai/together"
)

func TestRegisteredProviders(t *testing.T) {
	got := ai.RegisteredProviders("")
	want := []string{"anthropic", "atlascloud", "gemini", "groq", "mistral", "openai", "together"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RegisteredProviders() = %#v, want %#v", got, want)
	}

	got = ai.RegisteredProviders("image")
	want = []string{"atlascloud", "openai"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RegisteredProviders(image) = %#v, want %#v", got, want)
	}

	got = ai.RegisteredProviders("video")
	want = []string{"atlascloud"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RegisteredProviders(video) = %#v, want %#v", got, want)
	}

	got = ai.RegisteredProviders("stream")
	want = []string{"atlascloud", "groq", "mistral", "openai", "together"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RegisteredProviders(stream) = %#v, want %#v", got, want)
	}
}

func TestCapabilityRows(t *testing.T) {
	got := ai.CapabilityRows()
	want := []ai.CapabilityRow{
		{Provider: "anthropic", Capabilities: ai.Capabilities{Model: true}},
		{Provider: "atlascloud", Capabilities: ai.Capabilities{Model: true, Image: true, Video: true, Stream: true}},
		{Provider: "gemini", Capabilities: ai.Capabilities{Model: true}},
		{Provider: "groq", Capabilities: ai.Capabilities{Model: true, Stream: true}},
		{Provider: "mistral", Capabilities: ai.Capabilities{Model: true, Stream: true}},
		{Provider: "openai", Capabilities: ai.Capabilities{Model: true, Image: true, Stream: true}},
		{Provider: "together", Capabilities: ai.Capabilities{Model: true, Stream: true}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CapabilityRows() = %#v, want %#v", got, want)
	}
}

func TestCapabilityMatrix(t *testing.T) {
	matrix := ai.CapabilityMatrix()

	for _, provider := range []string{"anthropic", "atlascloud", "gemini", "groq", "mistral", "openai", "together"} {
		caps, ok := matrix[provider]
		if !ok {
			t.Fatalf("CapabilityMatrix missing %q", provider)
		}
		if !caps.Model {
			t.Fatalf("CapabilityMatrix(%s).Model = false, want true", provider)
		}
	}

	if caps := ai.ProviderCapabilities("openai"); caps != (ai.Capabilities{Model: true, Image: true, Stream: true}) {
		t.Fatalf("ProviderCapabilities(openai) = %#v", caps)
	}
	if caps := ai.ProviderCapabilities("atlascloud"); caps != (ai.Capabilities{Model: true, Image: true, Video: true, Stream: true}) {
		t.Fatalf("ProviderCapabilities(atlascloud) = %#v", caps)
	}
	if caps := ai.ProviderCapabilities("missing"); caps != (ai.Capabilities{}) {
		t.Fatalf("ProviderCapabilities(missing) = %#v", caps)
	}
}

func TestRegisterStream(t *testing.T) {
	ai.RegisterStream("test-stream")

	if caps := ai.ProviderCapabilities("test-stream"); caps != (ai.Capabilities{Stream: true}) {
		t.Fatalf("ProviderCapabilities(test-stream) = %#v", caps)
	}

	got := ai.RegisteredProviders("stream")
	want := []string{"atlascloud", "groq", "mistral", "openai", "test-stream", "together"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RegisteredProviders(stream) = %#v, want %#v", got, want)
	}
}
