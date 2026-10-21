package gemini

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-micro.dev/v6/model"
)

func TestProvider_StreamReportsUsageAndSkipsThoughts(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"candidates\":[{\"content\":{\"parts\":[{\"thought\":true,\"text\":\"pondering\"},{\"text\":\"hello\"}]}}],\"usageMetadata\":{\"promptTokenCount\":3,\"candidatesTokenCount\":2,\"thoughtsTokenCount\":4,\"totalTokenCount\":9}}\n\n"))
	}))
	defer ts.Close()

	p := NewProvider(model.WithAPIKey("k"), model.WithBaseURL(ts.URL))
	stream, err := p.Stream(context.Background(), &model.Request{Prompt: "hi"})
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}
	defer stream.Close()

	chunk, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv returned error: %v", err)
	}
	if chunk.Reply != "hello" {
		t.Fatalf("Reply = %q, want hello without the thought", chunk.Reply)
	}
	want := model.Usage{InputTokens: 3, OutputTokens: 6, TotalTokens: 9}
	if chunk.Usage != want {
		t.Fatalf("Usage = %+v, want %+v", chunk.Usage, want)
	}
}

func TestProvider_GenerateReportsUsage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"thought":true,"text":"pondering"},{"text":"hello"}]}}],"usageMetadata":{"promptTokenCount":3,"candidatesTokenCount":2,"thoughtsTokenCount":4,"totalTokenCount":9}}`))
	}))
	defer ts.Close()

	p := NewProvider(model.WithAPIKey("k"), model.WithBaseURL(ts.URL))
	resp, err := p.Generate(context.Background(), &model.Request{Prompt: "hi"})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if resp.Reply != "hello" {
		t.Fatalf("Reply = %q, want hello", resp.Reply)
	}
	want := model.Usage{InputTokens: 3, OutputTokens: 6, TotalTokens: 9}
	if resp.Usage != want {
		t.Fatalf("Usage = %+v, want %+v", resp.Usage, want)
	}
}

func TestProvider_GenerateSumsUsageOverToolRound(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"functionCall":{"id":"c1","name":"lookup","args":{}}}]}}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5,"totalTokenCount":15}}`))
			return
		}
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"done"}]}}],"usageMetadata":{"promptTokenCount":20,"candidatesTokenCount":7,"totalTokenCount":27}}`))
	}))
	defer ts.Close()

	handler := func(ctx context.Context, tc model.ToolCall) model.ToolResult {
		return model.ToolResult{ID: tc.ID, Value: map[string]any{"ok": true}}
	}
	p := NewProvider(model.WithAPIKey("k"), model.WithBaseURL(ts.URL), model.WithToolHandler(handler))
	resp, err := p.Generate(context.Background(), &model.Request{
		Prompt: "hi",
		Tools:  []model.Tool{{Name: "lookup", Description: "look something up"}},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	want := model.Usage{InputTokens: 30, OutputTokens: 12, TotalTokens: 42}
	if resp.Usage != want {
		t.Fatalf("Usage = %+v, want %+v", resp.Usage, want)
	}
}
