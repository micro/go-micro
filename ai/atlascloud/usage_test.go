package atlascloud

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-micro.dev/v6/ai"
)

// Generate reports what the completion cost, the same as Stream does.
//
// ai.Response has carried a Usage field from the start, and only the streaming
// path filled it in — the final chunk after include_usage. The plain path
// parsed choices and nothing else, so the API returned token counts on every
// completion and the struct never asked for them.
//
// The two paths disagreeing is the whole bug: a caller metering spend got real
// numbers from a stream and zeroes from Generate, and a zero is
// indistinguishable from a call that cost nothing. An agent runs on Generate,
// so the largest consumer of tokens was the one reporting none.
func TestGenerateReportsUsage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices":[{"message":{"content":"hello"}}],
			"usage":{"prompt_tokens":7,"completion_tokens":2,"total_tokens":9}
		}`))
	}))
	defer ts.Close()

	p := NewProvider(ai.WithAPIKey("test-key"), ai.WithBaseURL(ts.URL))
	resp, err := p.Generate(context.Background(), &ai.Request{Prompt: "Hello"})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if resp.Reply != "hello" {
		t.Errorf("Reply = %q, want hello", resp.Reply)
	}
	want := ai.Usage{InputTokens: 7, OutputTokens: 2, TotalTokens: 9}
	if resp.Usage != want {
		t.Errorf("Usage = %+v, want %+v — a caller metering spend cannot tell a\n"+
			"call that reported nothing from one that cost nothing", resp.Usage, want)
	}
}

// A response with no usage in it is still a response. Not every deployment
// returns the block, and a missing count is zero rather than an error.
func TestGenerateWithoutUsageStillAnswers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"hello"}}]}`))
	}))
	defer ts.Close()

	p := NewProvider(ai.WithAPIKey("test-key"), ai.WithBaseURL(ts.URL))
	resp, err := p.Generate(context.Background(), &ai.Request{Prompt: "Hello"})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if resp.Reply != "hello" {
		t.Errorf("Reply = %q, want hello", resp.Reply)
	}
	if resp.Usage != (ai.Usage{}) {
		t.Errorf("Usage = %+v, want zero", resp.Usage)
	}
}
