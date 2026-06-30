package ai_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"go-micro.dev/v6/ai"
	_ "go-micro.dev/v6/ai/anthropic"
	_ "go-micro.dev/v6/ai/atlascloud"
	_ "go-micro.dev/v6/ai/gemini"
	_ "go-micro.dev/v6/ai/groq"
	_ "go-micro.dev/v6/ai/mistral"
	_ "go-micro.dev/v6/ai/openai"
	_ "go-micro.dev/v6/ai/together"
)

func TestStreamProvidersConformToOpenAICompatibleSSE(t *testing.T) {
	providers := conformingStreamProviders(t)

	for _, provider := range providers {
		provider := provider
		t.Run(provider, func(t *testing.T) {
			var sawRequest bool
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sawRequest = true
				if r.URL.Path != "/v1/chat/completions" {
					t.Fatalf("path = %s, want /v1/chat/completions", r.URL.Path)
				}
				if got := r.Header.Get("Accept"); got != "text/event-stream" {
					t.Fatalf("Accept = %q, want text/event-stream", got)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
					t.Fatalf("Authorization = %q, want bearer API key", got)
				}

				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				if body["model"] == "" {
					t.Fatal("request omitted model")
				}
				if body["stream"] != true {
					t.Fatalf("stream = %#v, want true", body["stream"])
				}
				streamOptions, ok := body["stream_options"].(map[string]any)
				if !ok || streamOptions["include_usage"] != true {
					t.Fatalf("stream_options = %#v, want include_usage=true", body["stream_options"])
				}
				messages, ok := body["messages"].([]any)
				if !ok || len(messages) != 4 {
					t.Fatalf("messages = %#v, want system + history + prompt", body["messages"])
				}
				wantRoles := []string{"system", "user", "assistant", "user"}
				for i, wantRole := range wantRoles {
					message, ok := messages[i].(map[string]any)
					if !ok || message["role"] != wantRole {
						t.Fatalf("message[%d] = %#v, want role %q", i, messages[i], wantRole)
					}
				}

				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = w.Write([]byte(": keepalive\n\n"))
				_, _ = w.Write([]byte("event: ignored\n\n"))
				_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hel\"}}]}\n\n"))
				_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}]}\n\n"))
				_, _ = w.Write([]byte("data: {\"choices\":[],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":2,\"total_tokens\":5}}\n\n"))
				_, _ = w.Write([]byte("data: [DONE]\n\n"))
			}))
			defer ts.Close()

			model := ai.New(provider, ai.WithAPIKey("test-key"), ai.WithBaseURL(ts.URL))
			if model == nil {
				t.Fatalf("ai.New(%q) returned nil", provider)
			}
			stream, err := model.Stream(context.Background(), &ai.Request{
				SystemPrompt: "system",
				Messages: []ai.Message{
					{Role: "user", Content: "previous question"},
					{Role: "assistant", Content: "previous answer"},
				},
				Prompt: "current question",
			})
			if err != nil {
				t.Fatalf("Stream returned error: %v", err)
			}
			defer stream.Close()
			if !sawRequest {
				t.Fatal("server did not receive stream request")
			}

			assertStreamReply(t, stream, "hel")
			assertStreamReply(t, stream, "lo")
			usage, err := stream.Recv()
			if err != nil {
				t.Fatalf("usage chunk error: %v", err)
			}
			if usage.Reply != "" || usage.Usage != (ai.Usage{InputTokens: 3, OutputTokens: 2, TotalTokens: 5}) {
				t.Fatalf("usage chunk = %#v", usage)
			}
			if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
				t.Fatalf("final error = %v, want EOF", err)
			}
		})
	}
}

func TestStreamProvidersCloseCancelsInFlightRequest(t *testing.T) {
	for _, provider := range conformingStreamProviders(t) {
		provider := provider
		t.Run(provider, func(t *testing.T) {
			released := make(chan struct{})
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hel\"}}]}\n\n"))
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
				<-r.Context().Done()
				close(released)
			}))
			defer ts.Close()

			stream, err := ai.New(provider, ai.WithAPIKey("test-key"), ai.WithBaseURL(ts.URL)).Stream(context.Background(), &ai.Request{Prompt: "Hello"})
			if err != nil {
				t.Fatalf("Stream returned error: %v", err)
			}
			assertStreamReply(t, stream, "hel")
			if err := stream.Close(); err != nil {
				t.Fatalf("Close returned error: %v", err)
			}
			if err := stream.Close(); err != nil {
				t.Fatalf("second Close returned error: %v", err)
			}

			select {
			case <-released:
			case <-time.After(time.Second):
				t.Fatal("server did not observe canceled stream request")
			}
		})
	}
}

func TestUnsupportedProvidersReturnStreamingUnsupportedAndStayUnregistered(t *testing.T) {
	for _, provider := range []string{"anthropic", "gemini"} {
		provider := provider
		t.Run(provider, func(t *testing.T) {
			if caps := ai.ProviderCapabilities(provider); caps.Stream {
				t.Fatalf("ProviderCapabilities(%q).Stream = true, want false", provider)
			}
			_, err := ai.New(provider, ai.WithAPIKey("test-key")).Stream(context.Background(), &ai.Request{Prompt: "Hello"})
			if !errors.Is(err, ai.ErrStreamingUnsupported) {
				t.Fatalf("Stream error = %v, want ErrStreamingUnsupported", err)
			}
			if err != nil && strings.Contains(err.Error(), "test-key") {
				t.Fatal("streaming unsupported error leaked API key")
			}
		})
	}
}

func conformingStreamProviders(t *testing.T) []string {
	t.Helper()
	providers := ai.RegisteredProviders("stream")
	allowed := map[string]struct{}{
		"atlascloud": {},
		"groq":       {},
		"mistral":    {},
		"openai":     {},
		"together":   {},
	}
	var out []string
	for _, provider := range providers {
		if _, ok := allowed[provider]; ok {
			out = append(out, provider)
		}
	}
	want := []string{"atlascloud", "groq", "mistral", "openai", "together"}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("conforming stream providers = %#v, want %#v (registered stream providers: %#v)", out, want, providers)
	}
	return out
}

func assertStreamReply(t *testing.T, stream ai.Stream, want string) {
	t.Helper()
	chunk, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv error = %v, want reply %q", err, want)
	}
	if chunk.Reply != want {
		t.Fatalf("Reply = %q, want %q", chunk.Reply, want)
	}
}
