package ollama

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-micro.dev/v6/model"
)

type endpointRoundTripper func(*http.Request) (*http.Response, error)

func (f endpointRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestProviderChatEndpoints(t *testing.T) {
	for _, tc := range []struct {
		name    string
		baseURL string
		path    string
		cloud   bool
	}{
		{"cloud_origin", "https://ollama.com", "/v1/chat/completions", true},
		{"cloud_api_base", "https://ollama.com/v1", "/v1/chat/completions", true},
		{"cloud_api_base_slash", "https://ollama.com/v1/", "/v1/chat/completions", true},
		{"native_origin", "http://localhost:11434", "/api/chat", false},
		{"native_origin_slash", "http://localhost:11434/", "/api/chat", false},
		{"native_prefix", "http://localhost:11434/proxy", "/proxy/api/chat", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, streaming := range []bool{false, true} {
				name := "generate"
				if streaming {
					name = "stream"
				}
				t.Run(name, func(t *testing.T) {
					handler := http.NewServeMux()
					handler.HandleFunc(tc.path, func(w http.ResponseWriter, r *http.Request) {
						if streaming {
							if tc.cloud {
								w.Header().Set("Content-Type", "text/event-stream")
								_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"done\"}}]}\n\ndata: [DONE]\n\n")
							} else {
								w.Header().Set("Content-Type", "application/x-ndjson")
								_, _ = io.WriteString(w, "{\"message\":{\"content\":\"done\"},\"done\":false}\n{\"done\":true}\n")
							}
						} else {
							w.Header().Set("Content-Type", "application/json")
							if tc.cloud {
								_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"done"}}]}`)
							} else {
								_, _ = io.WriteString(w, `{"message":{"content":"done"},"done":true}`)
							}
						}
					})
					originalClient := http.DefaultClient
					http.DefaultClient = &http.Client{Transport: endpointRoundTripper(func(r *http.Request) (*http.Response, error) {
						if r.URL.Path != tc.path {
							t.Errorf("request path = %q, want %q", r.URL.Path, tc.path)
						}
						recorder := httptest.NewRecorder()
						handler.ServeHTTP(recorder, r)
						response := recorder.Result()
						response.Request = r
						return response, nil
					})}
					t.Cleanup(func() { http.DefaultClient = originalClient })

					p := NewProvider(model.WithBaseURL(tc.baseURL), model.WithAPIKey("test-key"))
					request := &model.Request{Prompt: "Hello"}
					if streaming {
						stream, err := p.Stream(context.Background(), request)
						if err != nil {
							t.Fatalf("Stream at %s: %v", tc.baseURL, err)
						}
						defer stream.Close()
						chunk, err := stream.Recv()
						if err != nil || chunk.Reply != "done" {
							t.Fatalf("Recv = %#v, %v", chunk, err)
						}
					} else {
						response, err := p.Generate(context.Background(), request)
						if err != nil {
							t.Fatalf("Generate at %s: %v", tc.baseURL, err)
						}
						if response.Reply != "done" {
							t.Fatalf("Reply = %q, want done", response.Reply)
						}
					}
				})
			}
		})
	}
}
