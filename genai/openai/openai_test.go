package openai

import (
	"os"
	"testing"
	"go-micro.dev/v5/genai"
)

func TestOpenAI_GenerateText(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}
	client := New(genai.WithAPIKey(apiKey))
	resp, err := client.GenerateText("Say hello world")
	if err != nil {
		t.Fatalf("GenerateText error: %v", err)
	}
	if resp == "" {
		t.Error("Expected non-empty text response")
	}
}

func TestOpenAI_GenerateImage(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}
	client := New(genai.WithAPIKey(apiKey))
	resp, err := client.GenerateImage("A cat wearing sunglasses")
	if err != nil {
		t.Fatalf("GenerateImage error: %v", err)
	}
	if resp == "" {
		t.Error("Expected non-empty image URL")
	}
}

