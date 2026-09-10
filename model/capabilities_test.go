package model_test

import (
	"reflect"
	"testing"

	"go-micro.dev/v6/model"
	_ "go-micro.dev/v6/model/anthropic"
	_ "go-micro.dev/v6/model/atlascloud"
	_ "go-micro.dev/v6/model/gemini"
	_ "go-micro.dev/v6/model/groq"
	_ "go-micro.dev/v6/model/minimax"
	_ "go-micro.dev/v6/model/mistral"
	_ "go-micro.dev/v6/model/openai"
	_ "go-micro.dev/v6/model/together"
)

func TestRegisteredProviders(t *testing.T) {
	got := model.RegisteredProviders("")
	want := []string{"anthropic", "atlascloud", "gemini", "groq", "minimax", "mistral", "openai", "together"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RegisteredProviders() = %#v, want %#v", got, want)
	}

	got = model.RegisteredProviders("image")
	want = []string{"atlascloud", "openai"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RegisteredProviders(image) = %#v, want %#v", got, want)
	}

	got = model.RegisteredProviders("video")
	want = []string{"atlascloud"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RegisteredProviders(video) = %#v, want %#v", got, want)
	}

	got = model.RegisteredProviders("stream")
	want = []string{"anthropic", "atlascloud", "gemini", "groq", "minimax", "mistral", "openai", "together"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RegisteredProviders(stream) = %#v, want %#v", got, want)
	}
}

func TestCapabilityRows(t *testing.T) {
	got := model.CapabilityRows()
	want := []model.CapabilityRow{
		{Provider: "anthropic", Capabilities: model.Capabilities{Model: true, Stream: true, ToolStream: true}},
		{Provider: "atlascloud", Capabilities: model.Capabilities{Model: true, Image: true, Video: true, Stream: true}},
		{Provider: "gemini", Capabilities: model.Capabilities{Model: true, Stream: true}},
		{Provider: "groq", Capabilities: model.Capabilities{Model: true, Stream: true, ToolStream: true}},
		{Provider: "minimax", Capabilities: model.Capabilities{Model: true, Stream: true, ToolStream: true}},
		{Provider: "mistral", Capabilities: model.Capabilities{Model: true, Stream: true, ToolStream: true}},
		{Provider: "openai", Capabilities: model.Capabilities{Model: true, Image: true, Stream: true, ToolStream: true}},
		{Provider: "together", Capabilities: model.Capabilities{Model: true, Stream: true, ToolStream: true}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CapabilityRows() = %#v, want %#v", got, want)
	}
}

func TestCapabilityMatrix(t *testing.T) {
	matrix := model.CapabilityMatrix()

	for _, provider := range []string{"anthropic", "atlascloud", "gemini", "groq", "minimax", "mistral", "openai", "together"} {
		caps, ok := matrix[provider]
		if !ok {
			t.Fatalf("CapabilityMatrix missing %q", provider)
		}
		if !caps.Model {
			t.Fatalf("CapabilityMatrix(%s).Model = false, want true", provider)
		}
	}

	if caps := model.ProviderCapabilities("openai"); caps != (model.Capabilities{Model: true, Image: true, Stream: true, ToolStream: true}) {
		t.Fatalf("ProviderCapabilities(openai) = %#v", caps)
	}
	if caps := model.ProviderCapabilities("atlascloud"); caps != (model.Capabilities{Model: true, Image: true, Video: true, Stream: true}) {
		t.Fatalf("ProviderCapabilities(atlascloud) = %#v", caps)
	}
	if caps := model.ProviderCapabilities("missing"); caps != (model.Capabilities{}) {
		t.Fatalf("ProviderCapabilities(missing) = %#v", caps)
	}
}

func TestRegisterStream(t *testing.T) {
	model.RegisterStream("test-stream")

	if caps := model.ProviderCapabilities("test-stream"); caps != (model.Capabilities{Stream: true}) {
		t.Fatalf("ProviderCapabilities(test-stream) = %#v", caps)
	}

	got := model.RegisteredProviders("stream")
	want := []string{"anthropic", "atlascloud", "gemini", "groq", "minimax", "mistral", "openai", "test-stream", "together"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RegisteredProviders(stream) = %#v, want %#v", got, want)
	}
}

func TestRegisterToolStream(t *testing.T) {
	model.RegisterToolStream("test-tool-stream")

	if caps := model.ProviderCapabilities("test-tool-stream"); caps != (model.Capabilities{ToolStream: true}) {
		t.Fatalf("ProviderCapabilities(test-tool-stream) = %#v", caps)
	}

	got := model.RegisteredProviders("tool_stream")
	want := []string{"anthropic", "groq", "minimax", "mistral", "openai", "test-tool-stream", "together"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RegisteredProviders(tool_stream) = %#v, want %#v", got, want)
	}
}
