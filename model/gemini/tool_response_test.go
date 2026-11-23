package gemini_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go-micro.dev/v6/model"
	"go-micro.dev/v6/model/gemini"
)

func TestGenerateToolResponseObjects(t *testing.T) {
	tests := []struct {
		name   string
		result model.ToolResult
		want   string
	}{
		{"custom text tool", model.ToolResult{Value: "found", Content: "found"}, `{"result":"found"}`},
		{"content only", model.ToolResult{Content: "Approval denied"}, `{"result":"Approval denied"}`},
		{"list result", model.ToolResult{Value: []string{"first", "second"}}, `{"result":["first","second"]}`},
		{"object result", model.ToolResult{Value: map[string]any{"count": 2}}, `{"count":2}`},
		{"error object", model.ToolResult{Value: map[string]string{"error": "denied"}}, `{"error":"denied"}`},
		{"struct result", model.ToolResult{Value: struct {
			Count int `json:"count"`
		}{Count: 2}}, `{"count":2}`},
	}
	for _, incremental := range []bool{false, true} {
		for _, tc := range tests {
			t.Run(fmt.Sprintf("%s/incremental_%t", tc.name, incremental), func(t *testing.T) {
				var want any
				if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
					t.Fatal(err)
				}
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var request struct {
						Contents []struct {
							Parts []struct {
								FunctionResponse *struct {
									Name     string          `json:"name"`
									ID       string          `json:"id"`
									Response json.RawMessage `json:"response"`
								} `json:"functionResponse"`
							} `json:"parts"`
						} `json:"contents"`
					}
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					call := calls.Add(1)
					response := `{"candidates":[{"content":{"parts":[{"functionCall":{"id":"call-1","name":"lookup","args":{}}}]},"finishReason":"STOP"}]}`
					if call > 1 {
						responses := 0
						for _, content := range request.Contents {
							for _, part := range content.Parts {
								fn := part.FunctionResponse
								if fn == nil {
									continue
								}
								responses++
								// Gemini declares response as google.protobuf.Struct: it must be an object.
								var object map[string]json.RawMessage
								if err := json.Unmarshal(fn.Response, &object); err != nil || object == nil {
									w.WriteHeader(http.StatusBadRequest)
									_, _ = io.WriteString(w, `{"error":{"message":"functionResponse.response must be a JSON object"}}`)
									return
								}
								var got any
								if err := json.Unmarshal(fn.Response, &got); err != nil {
									t.Error(err)
								}
								if !reflect.DeepEqual(got, want) {
									t.Errorf("response = %#v, want %#v", got, want)
								}
								if fn.Name != "lookup" || fn.ID != "call-1" {
									t.Errorf("lost call identity: %+v", fn)
								}
							}
						}
						if responses != 1 {
							t.Errorf("function responses = %d, want 1", responses)
						}
						response = `{"candidates":[{"content":{"parts":[{"text":"complete"}]},"finishReason":"STOP"}]}`
					}
					if incremental {
						w.Header().Set("Content-Type", "text/event-stream")
						_, _ = fmt.Fprintf(w, "data: %s\n\n", response)
					} else {
						w.Header().Set("Content-Type", "application/json")
						_, _ = io.WriteString(w, response)
					}
				}))
				defer server.Close()
				toolCalls := 0
				provider := gemini.NewProvider(model.WithBaseURL(server.URL), model.WithAPIKey("fixture-key"), model.WithModel("fixture"), model.WithToolHandler(func(_ context.Context, call model.ToolCall) model.ToolResult {
					toolCalls++
					if call.Name != "lookup" {
						t.Errorf("unexpected tool %q", call.Name)
					}
					return tc.result
				}))
				var opts []model.GenerateOption
				var tokens strings.Builder
				if incremental {
					opts = append(opts, model.WithTokenHandler(func(text string) { tokens.WriteString(text) }))
				}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				response, err := provider.Generate(ctx, &model.Request{Prompt: "Look it up", Tools: []model.Tool{{Name: "lookup"}}}, opts...)
				if err != nil {
					t.Fatal(err)
				}
				if response.Answer != "complete" {
					t.Fatalf("answer = %q", response.Answer)
				}
				if incremental && tokens.String() != "complete" {
					t.Fatalf("tokens = %q", tokens.String())
				}
				if calls.Load() != 2 || toolCalls != 1 {
					t.Fatalf("requests = %d, tools = %d", calls.Load(), toolCalls)
				}
			})
		}
	}
}
