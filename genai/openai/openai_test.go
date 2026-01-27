package openai

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"go-micro.dev/v5/genai"
)

func TestOpenAI_GenerateText(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	client := New(genai.WithAPIKey(apiKey))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := client.Generate(ctx, "Say hello world", genai.Text)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if res == nil || res.Text == "" {
		t.Error("Expected non-empty text response")
	}
}

func TestOpenAI_StreamText(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	client := New(genai.WithAPIKey(apiKey))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := client.Stream(ctx, "Count from 1 to 5", genai.Text)
	if err != nil {
		t.Fatalf("Stream error: %v", err)
	}
	defer stream.Close()

	var fullText strings.Builder
	chunkCount := 0
	for result := range stream.Results {
		if result.Error != nil {
			t.Fatalf("Stream chunk error: %v", result.Error)
		}
		fullText.WriteString(result.Text)
		chunkCount++
	}

	if chunkCount == 0 {
		t.Error("Expected at least one chunk")
	}
	if fullText.Len() == 0 {
		t.Error("Expected non-empty streamed response")
	}
}

func TestOpenAI_GenerateImage(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	client := New(genai.WithAPIKey(apiKey))
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	res, err := client.Generate(ctx, "A cat wearing sunglasses", genai.Image)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if res == nil || res.Text == "" {
		t.Error("Expected non-empty image URL")
	}
}

func TestOpenAI_ContextCancellation(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	client := New(genai.WithAPIKey(apiKey))

	// Create an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Generate(ctx, "Say hello", genai.Text)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
}
