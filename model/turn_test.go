package model_test

import (
	"context"
	"encoding/json"
	"go-micro.dev/v6/model"
	"go-micro.dev/v6/model/anthropic"
	"go-micro.dev/v6/model/atlascloud"
	"go-micro.dev/v6/model/gemini"
	"go-micro.dev/v6/model/groq"
	"go-micro.dev/v6/model/minimax"
	"go-micro.dev/v6/model/mistral"
	"go-micro.dev/v6/model/openai"
	"go-micro.dev/v6/model/together"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProviderTurnsPreserveContinuation(t *testing.T) {
	factories := map[string]func(...model.Option) model.Model{
		"openai":     func(o ...model.Option) model.Model { return openai.NewProvider(o...) },
		"groq":       func(o ...model.Option) model.Model { return groq.NewProvider(o...) },
		"mistral":    func(o ...model.Option) model.Model { return mistral.NewProvider(o...) },
		"together":   func(o ...model.Option) model.Model { return together.NewProvider(o...) },
		"minimax":    func(o ...model.Option) model.Model { return minimax.NewProvider(o...) },
		"atlascloud": func(o ...model.Option) model.Model { return atlascloud.NewProvider(o...) },
		"anthropic":  func(o ...model.Option) model.Model { return anthropic.NewProvider(o...) },
		"gemini":     func(o ...model.Option) model.Model { return gemini.NewProvider(o...) },
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			calls, executed := 0, 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				var req map[string]any
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
				}
				if calls == 2 {
					data, _ := json.Marshal(req)
					if !strings.Contains(string(data), "signature-to-preserve") || !strings.Contains(string(data), "tool-result") {
						t.Errorf("lost continuation: %s", data)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				switch name {
				case "anthropic":
					if calls == 1 {
						_, _ = w.Write([]byte(`{"content":[{"type":"thinking","thinking":"reasoning","signature":"signature-to-preserve"},{"type":"tool_use","id":"call1","name":"lookup","input":{}}],"stop_reason":"tool_use","usage":{"input_tokens":2,"output_tokens":3}}`))
					} else {
						_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"done"}],"stop_reason":"end_turn"}`))
					}
				case "gemini":
					if calls == 1 {
						_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"thoughtSignature":"signature-to-preserve","functionCall":{"id":"call1","name":"lookup","args":{}}}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":2,"candidatesTokenCount":3,"totalTokenCount":5}}`))
					} else {
						_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"done"}]},"finishReason":"STOP"}]}`))
					}
				default:
					if calls == 1 {
						_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"","reasoning_content":"signature-to-preserve","tool_calls":[{"id":"call1","type":"function","function":{"name":"lookup","arguments":"{}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}}`))
					} else {
						_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"done"},"finish_reason":"stop"}]}`))
					}
				}
			}))
			defer srv.Close()
			provider := factory(model.WithBaseURL(srv.URL), model.WithAPIKey("test"), model.WithModel("test"), model.WithToolHandler(func(context.Context, model.ToolCall) model.ToolResult { executed++; return model.ToolResult{} }))
			turn := provider.(model.Turner)
			first, err := turn.Turn(context.Background(), &model.Request{Prompt: "lookup", Tools: []model.Tool{{Name: "lookup"}}})
			if err != nil {
				t.Fatal(err)
			}
			if executed != 0 || calls != 1 || len(first.ToolCalls) != 1 || first.Usage.TotalTokens != 5 {
				t.Fatalf("invalid first turn: %+v calls=%d executed=%d", first, calls, executed)
			}
			first.Continuation.Results = []model.ToolResult{{ID: "call1", Name: "lookup", Content: "tool-result"}}
			last, err := turn.Turn(context.Background(), &model.Request{Continuation: first.Continuation, Tools: []model.Tool{{Name: "lookup"}}})
			if err != nil {
				t.Fatal(err)
			}
			if last.Reply != "done" || last.StopReason == "" || calls != 2 || executed != 0 {
				t.Fatalf("invalid final turn: %+v", last)
			}
		})
	}
}
