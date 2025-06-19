package openai

import (
	"os"
	"testing"
	"workspaces/go-micro/genai"
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

func TestOpenAI_SpeechToText(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}
	// This test requires a valid audio file in wav format.
	f, err := os.Open("testdata/hello.wav")
	if err != nil {
		t.Skip("testdata/hello.wav not found")
	}
	defer f.Close()
	data := make([]byte, 0, 1024*1024)
	buf := make([]byte, 4096)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	client := New(genai.WithAPIKey(apiKey))
	resp, err := client.SpeechToText(data)
	if err != nil {
		t.Fatalf("SpeechToText error: %v", err)
	}
	if resp == "" {
		t.Error("Expected non-empty speech-to-text response")
	}
}
