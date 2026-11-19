package openaiapi_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-micro.dev/v6/model"
	_ "go-micro.dev/v6/model/atlascloud"
	_ "go-micro.dev/v6/model/groq"
	_ "go-micro.dev/v6/model/litellm"
	_ "go-micro.dev/v6/model/minimax"
	_ "go-micro.dev/v6/model/mistral"
	_ "go-micro.dev/v6/model/openai"
	_ "go-micro.dev/v6/model/openrouter"
	_ "go-micro.dev/v6/model/together"
)

func TestStreamProvidersPropagateSSEErrors(t *testing.T) {
	providers := []string{"atlascloud", "groq", "litellm", "minimax", "mistral", "openai", "openrouter", "together"}
	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			for _, afterContent := range []bool{false, true} {
				name := "before_content"
				if afterContent {
					name = "after_content"
				}
				t.Run(name, func(t *testing.T) {
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "text/event-stream")
						if afterContent {
							_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n")
							_, _ = io.WriteString(w, "data: {\"error\":{\"message\":\"upstream overloaded\",\"code\":502},\"choices\":[{\"delta\":{},\"finish_reason\":\"error\"}]}\n\n")
						} else {
							_, _ = io.WriteString(w, "data: {\"error\":{\"message\":\"upstream overloaded\",\"code\":502}}\n\n")
						}
						_, _ = io.WriteString(w, "data: [DONE]\n\n")
					}))
					defer server.Close()

					m := model.New(provider, model.WithAPIKey("test-key"), model.WithBaseURL(server.URL))
					stream, err := m.Stream(context.Background(), &model.Request{Prompt: "Hello"})
					if err != nil {
						t.Fatalf("Stream returned error before reading the SSE event: %v", err)
					}
					defer stream.Close()
					if afterContent {
						chunk, err := stream.Recv()
						if err != nil || chunk.Reply != "partial" {
							t.Fatalf("first chunk = %#v, %v; want partial", chunk, err)
						}
					}
					response, err := stream.Recv()
					if response != nil || err == nil || errors.Is(err, io.EOF) || !strings.Contains(err.Error(), "upstream overloaded") {
						t.Fatalf("Recv = %#v, %v; want the provider error", response, err)
					}
				})
			}
		})
	}
}
