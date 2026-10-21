package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-micro.dev/v6/model"
)

func TestProvider_GenerateInvalidToolArguments(t *testing.T) {
	tests := []struct {
		name      string
		arguments string // JSON string literal as it appears in the response body
		wantErr   bool
	}{
		{name: "truncated JSON fails", arguments: `"{\"a\":"`, wantErr: true},
		{name: "empty string means no arguments", arguments: `""`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"tool_calls","message":{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"add","arguments":` + tt.arguments + `}}]}}]}`))
			}))
			defer ts.Close()

			p := NewProvider(model.WithAPIKey("test-key"), model.WithBaseURL(ts.URL))
			resp, err := p.Generate(context.Background(), &model.Request{
				Prompt: "add 1 and 2",
				Tools:  []model.Tool{{Name: "add", Description: "Add numbers", Properties: map[string]any{"a": map[string]any{"type": "number"}}}},
			})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Generate returned no error for truncated tool arguments, tool calls: %+v", resp.ToolCalls)
				}
				if !strings.Contains(err.Error(), "tool add arguments") {
					t.Fatalf("error = %v, want it to name the tool and its arguments", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}
			if len(resp.ToolCalls) != 1 || len(resp.ToolCalls[0].Input) != 0 {
				t.Fatalf("tool calls = %+v, want one call with empty input", resp.ToolCalls)
			}
		})
	}
}
