package groq

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-micro.dev/v6/ai"
)

func TestProvider_String(t *testing.T) {
	if NewProvider().String() != "groq" {
		t.Errorf("got %q", NewProvider().String())
	}
}

func TestProvider_Defaults(t *testing.T) {
	opts := NewProvider().Options()
	if opts.Model != "llama-3.3-70b-versatile" {
		t.Errorf("default model = %q", opts.Model)
	}
	if opts.BaseURL != "https://api.groq.com/openai" {
		t.Errorf("default base URL = %q", opts.BaseURL)
	}
}

func TestProvider_Init(t *testing.T) {
	p := NewProvider()
	if err := p.Init(ai.WithModel("m"), ai.WithAPIKey("k")); err != nil {
		t.Fatal(err)
	}
	if p.Options().Model != "m" || p.Options().APIKey != "k" {
		t.Error("Init did not apply options")
	}
}

func TestProvider_Generate_NoAPIKey(t *testing.T) {
	if _, err := NewProvider().Generate(context.Background(), &ai.Request{Prompt: "hi"}); err == nil {
		t.Error("expected error without API key")
	}
}

func TestProvider_Stream(t *testing.T) {
	var sawStream bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %s, want /v1/chat/completions", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		sawStream, _ = body["stream"].(bool)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hel\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer ts.Close()

	p := NewProvider(ai.WithAPIKey("test-key"), ai.WithBaseURL(ts.URL))
	stream, err := p.Stream(context.Background(), &ai.Request{Prompt: "Hello"})
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}
	defer stream.Close()
	if !sawStream {
		t.Fatal("stream request did not set stream=true")
	}

	first, err := stream.Recv()
	if err != nil || first.Reply != "hel" {
		t.Fatalf("first chunk = %#v, %v; want hel", first, err)
	}
	second, err := stream.Recv()
	if err != nil || second.Reply != "lo" {
		t.Fatalf("second chunk = %#v, %v; want lo", second, err)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("final error = %v, want EOF", err)
	}
}

func TestProvider_Registration(t *testing.T) {
	m := ai.New("groq", ai.WithAPIKey("test"))
	if m == nil {
		t.Fatal("provider not registered")
	}
	if m.String() != "groq" {
		t.Errorf("got %q", m.String())
	}
}
