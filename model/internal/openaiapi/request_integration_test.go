package openaiapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-micro.dev/v6/model"
	"go-micro.dev/v6/model/groq"
	"go-micro.dev/v6/model/openai"
)

func TestChatRequestParity(t *testing.T) {
	for name, factory := range map[string]func(...model.Option) model.Model{"groq": func(opts ...model.Option) model.Model { return groq.NewProvider(opts...) }, "openai": func(opts ...model.Option) model.Model { return openai.NewProvider(opts...) }} {
		for _, mode := range []string{"generate", "followup_error", "stream"} {
			t.Run(name+"/"+mode, func(t *testing.T) {
				requests, calls := 0, 0
				ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests++
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
						return
					}
					if body["max_tokens"] != float64(1024) || body["reasoning_effort"] != "high" || body["model"] != "test" || body["temperature"] != float64(0) {
						t.Errorf("lost options: %v", body)
					}
					messages := body["messages"].([]any)
					for i, want := range []string{"system", "earlier question", "earlier answer", "next question"} {
						if len(messages) <= i || messages[i].(map[string]any)["content"] != want {
							t.Errorf("lost history: %v", messages)
							return
						}
					}
					if mode == "stream" {
						if body["stream"] != true {
							t.Error("stream flag missing")
						}
						w.Header().Set("Content-Type", "text/event-stream")
						fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"done\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
						return
					}
					if len(body["tools"].([]any)) != 1 {
						t.Error("tools missing")
					}
					if requests == 1 {
						fmt.Fprint(w, `{"choices":[{"message":{"content":"","tool_calls":[{"id":"call1","type":"function","function":{"name":"lookup","arguments":"{}"}}]}}]}`)
						return
					}
					if len(messages) != 6 {
						t.Errorf("messages: %v", messages)
						return
					}
					assistant := messages[4].(map[string]any)
					call := assistant["tool_calls"].([]any)[0].(map[string]any)
					if call["type"] != "function" {
						t.Errorf("lost tool type: %v", call)
					}
					tool := messages[5].(map[string]any)
					if tool["tool_call_id"] != "call1" || tool["content"] != "result" {
						t.Errorf("lost result: %v", tool)
					}
					if mode == "followup_error" {
						w.WriteHeader(http.StatusTooManyRequests)
						fmt.Fprint(w, `{"error":{"message":"slow down"}}`)
						return
					}
					fmt.Fprint(w, `{"choices":[{"message":{"content":"done"}}]}`)
				}))
				defer ts.Close()
				p := factory(model.WithBaseURL(ts.URL), model.WithAPIKey("test"), model.WithModel("test"), model.WithMaxTokens(1024), model.WithTemperature(0), model.WithEffort("high"), model.WithToolHandler(func(_ context.Context, c model.ToolCall) model.ToolResult {
					calls++
					return model.ToolResult{ID: c.ID, Content: "result"}
				}))
				req := &model.Request{SystemPrompt: "system", Prompt: "next question", Messages: []model.Message{{Role: "user", Content: "earlier question"}, {Role: "assistant", Content: "earlier answer"}}, Tools: []model.Tool{{Name: "lookup", Properties: map[string]any{}}}}
				if mode == "stream" {
					s, err := p.Stream(context.Background(), req)
					if err != nil {
						t.Fatal(err)
					}
					defer s.Close()
					var answer string
					for {
						chunk, err := s.Recv()
						if err == io.EOF {
							break
						}
						if err != nil {
							t.Fatal(err)
						}
						answer += chunk.Reply
					}
					if answer != "done" || requests != 1 {
						t.Fatalf("answer=%q requests=%d", answer, requests)
					}
					return
				}
				response, err := p.Generate(context.Background(), req)
				if mode == "followup_error" {
					var status interface{ StatusCode() int }
					if !errors.As(err, &status) || status.StatusCode() != 429 {
						t.Fatalf("lost provider error: %v", err)
					}
				} else if err != nil || response.Answer != "done" {
					t.Fatalf("response=%+v err=%v", response, err)
				}
				if requests != 2 || calls != 1 {
					t.Fatalf("requests=%d calls=%d", requests, calls)
				}
			})
		}
	}
}

func TestChatEmptyOutputLimit(t *testing.T) {
	for name, factory := range map[string]func(...model.Option) model.Model{
		"groq":   func(opts ...model.Option) model.Model { return groq.NewProvider(opts...) },
		"openai": func(opts ...model.Option) model.Model { return openai.NewProvider(opts...) },
	} {
		for _, streaming := range []bool{false, true} {
			for _, visible := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/stream=%v/visible=%v", name, streaming, visible), func(t *testing.T) {
					ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						content := ""
						if visible {
							content = "partial"
						}
						if streaming {
							fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q}}]}\n\n", content)
							fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"length\"}]}\n\ndata: [DONE]\n\n")
						} else {
							fmt.Fprintf(w, `{"choices":[{"message":{"content":%q},"finish_reason":"length"}]}`, content)
						}
					}))
					defer ts.Close()
					p := factory(model.WithBaseURL(ts.URL), model.WithAPIKey("test"))
					var err error
					if streaming {
						var s model.Stream
						s, err = p.Stream(context.Background(), &model.Request{Prompt: "hello"})
						if err != nil {
							t.Fatal(err)
						}
						defer s.Close()
						for {
							_, err = s.Recv()
							if err != nil {
								break
							}
						}
						if err == io.EOF {
							err = nil
						}
					} else {
						_, err = p.Generate(context.Background(), &model.Request{Prompt: "hello"})
					}
					if visible {
						if err != nil {
							t.Fatal(err)
						}
					} else if !errors.Is(err, model.ErrOutputLimit) {
						t.Fatalf("error=%v", err)
					}
				})
			}
		}
	}
}
