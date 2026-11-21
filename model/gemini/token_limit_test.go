package gemini_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"go-micro.dev/v6/model"
	"go-micro.dev/v6/model/gemini"
)

func TestMaxTokensAcrossRequestPaths(t *testing.T) {
	for _, mode := range []string{"generate", "generate_incremental", "generate_tools", "generate_incremental_tools", "stream"} {
		for _, limit := range []int{0, 32} {
			t.Run(fmt.Sprintf("%s/limit_%d", mode, limit), func(t *testing.T) {
				withTools := strings.HasSuffix(mode, "_tools")
				incremental := strings.Contains(mode, "incremental") || mode == "stream"
				var requests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					call := requests.Add(1)
					var request map[string]json.RawMessage
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					if limit == 0 {
						if _, ok := request["generationConfig"]; ok {
							t.Error("zero limit must leave the provider default unchanged")
						}
					} else {
						var config struct {
							MaxOutputTokens int `json:"maxOutputTokens"`
						}
						if err := json.Unmarshal(request["generationConfig"], &config); err != nil {
							t.Errorf("request %d has no generation config: %v", call, err)
						} else if config.MaxOutputTokens != limit {
							t.Errorf("request %d maxOutputTokens = %d, want %d", call, config.MaxOutputTokens, limit)
						}
					}
					suffix := ":generateContent"
					if incremental {
						suffix = ":streamGenerateContent"
					}
					if !strings.HasSuffix(r.URL.Path, suffix) {
						t.Errorf("unexpected endpoint %s", r.URL.Path)
					}
					response := `{"candidates":[{"content":{"parts":[{"text":"complete"}]},"finishReason":"STOP"}]}`
					if withTools && call == 1 {
						response = `{"candidates":[{"content":{"parts":[{"functionCall":{"id":"call","name":"lookup","args":{}}}]},"finishReason":"STOP"}]}`
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
				toolsRun := 0
				provider := gemini.NewProvider(
					model.WithAPIKey("fixture-key"), model.WithBaseURL(server.URL),
					model.WithModel("fixture"), model.WithMaxTokens(limit),
					model.WithToolHandler(func(_ context.Context, call model.ToolCall) model.ToolResult {
						toolsRun++
						if call.Name != "lookup" {
							t.Errorf("unexpected tool %s", call.Name)
						}
						return model.ToolResult{Value: map[string]any{"result": "found"}}
					}),
				)
				request := &model.Request{Prompt: "Look up the result"}
				if withTools {
					request.Tools = []model.Tool{{Name: "lookup", Properties: map[string]any{}}}
				}
				var text strings.Builder
				if mode == "stream" {
					stream, err := provider.Stream(context.Background(), request)
					if err != nil {
						t.Fatal(err)
					}
					defer stream.Close()
					for {
						response, err := stream.Recv()
						if err == io.EOF {
							break
						}
						if err != nil {
							t.Fatal(err)
						}
						text.WriteString(response.Reply)
					}
				} else {
					var opts []model.GenerateOption
					if incremental {
						opts = append(opts, model.WithTokenHandler(func(token string) { text.WriteString(token) }))
					}
					response, err := provider.Generate(context.Background(), request, opts...)
					if err != nil {
						t.Fatal(err)
					}
					if !incremental {
						if withTools {
							text.WriteString(response.Answer)
						} else {
							text.WriteString(response.Reply)
						}
					}
				}
				if text.String() != "complete" {
					t.Fatalf("reply = %q", text.String())
				}
				expectedCalls, expectedTools := int32(1), 0
				if withTools {
					expectedCalls, expectedTools = 2, 1
				}
				if requests.Load() != expectedCalls || toolsRun != expectedTools {
					t.Fatalf("requests=%d tools=%d, want %d/%d", requests.Load(), toolsRun, expectedCalls, expectedTools)
				}
			})
		}
	}
}
