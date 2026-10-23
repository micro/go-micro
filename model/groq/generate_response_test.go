package groq

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-micro.dev/v6/model"
)

func TestGenerateReportsUsageAndStopReason(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"length","message":{"content":"partial"}}],"usage":{"prompt_tokens":7,"completion_tokens":3,"total_tokens":10}}`))
	}))
	defer ts.Close()

	p := NewProvider(model.WithAPIKey("test-key"), model.WithBaseURL(ts.URL))
	resp, err := p.Generate(context.Background(), &model.Request{Prompt: "Hello"})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if want := (model.Usage{InputTokens: 7, OutputTokens: 3, TotalTokens: 10}); resp.Usage != want {
		t.Fatalf("usage = %+v, want %+v", resp.Usage, want)
	}
	if resp.StopReason != "length" {
		t.Fatalf("stop reason = %q, want length", resp.StopReason)
	}
}

func TestGenerateReturnsErrOutputLimitWithoutVisibleContent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"length","message":{"content":""}}],"usage":{"prompt_tokens":7,"completion_tokens":50,"total_tokens":57}}`))
	}))
	defer ts.Close()

	p := NewProvider(model.WithAPIKey("test-key"), model.WithBaseURL(ts.URL))
	_, err := p.Generate(context.Background(), &model.Request{Prompt: "Hello"})
	if !errors.Is(err, model.ErrOutputLimit) {
		t.Fatalf("err = %v, want model.ErrOutputLimit", err)
	}
}
