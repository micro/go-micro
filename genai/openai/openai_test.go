package openai

import (
	"go-micro.dev/v5/genai"
	"os"
	"testing"
)

func TestOpenAI_GenerateText(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}
	client := New(genai.WithAPIKey(apiKey))
	res, err := client.Generate("Say hello world", genai.Text)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if res == nil || res.Text == "" {
		t.Error("Expected non-empty text response")
	}
}

func TestOpenAI_GenerateImage(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}
	client := New(genai.WithAPIKey(apiKey))
	res, err := client.Generate("A cat wearing sunglasses", genai.Image)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if res == nil || res.Text == "" {
		t.Error("Expected non-empty image URL")
	}
}
