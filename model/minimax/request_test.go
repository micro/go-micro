package minimax

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"go-micro.dev/v6/model"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRequestBoundary(t *testing.T) {
	for _, mode := range []string{"generate", "stream", "token_callback"} {
		for _, endpoint := range []string{"", "https://api.minimax.cn", "https://custom.example"} {
			for _, test := range []struct {
				name, selected, effort, wantEffort, thinking string
				mode                                         model.ThinkingMode
			}{
				{name: "default", thinking: "adaptive"},
				{name: "flash-low", selected: "MiniMax-M3.1-Flash", effort: "low", wantEffort: "low", thinking: "adaptive"},
				{name: "flash-medium", effort: "medium", wantEffort: "medium", thinking: "adaptive"},
				{name: "flash-high", effort: "high", wantEffort: "high", thinking: "adaptive"},
				{name: "flash-xhigh", effort: "xhigh", wantEffort: "xhigh", thinking: "adaptive"},
				{name: "flash-max", effort: "max", wantEffort: "max", thinking: "adaptive", mode: model.ThinkingAdaptive},
				{name: "saved-m3", selected: "MiniMax-M3", effort: "high", mode: model.ThinkingDisabled, thinking: "disabled"},
				{name: "m3-adaptive", selected: "MiniMax-M3", mode: model.ThinkingAdaptive, thinking: "adaptive"},
				{name: "legacy", selected: "MiniMax-M2.7", effort: "high", wantEffort: "high"},
				{name: "custom", selected: "custom-model", effort: "none", wantEffort: "none"},
			} {
				t.Run(mode+"/"+endpoint+"/"+test.name, func(t *testing.T) {
					selected := test.selected
					if selected == "" {
						selected = "MiniMax-M3.1-Flash"
					}
					base := endpoint
					if base == "" {
						base = "https://api.minimax.io"
					}
					content := []any{
						map[string]any{"type": "image_url", "image_url": map[string]any{"url": "https://example.com/image.png"}},
						map[string]any{"type": "video_url", "video_url": map[string]any{"url": "https://example.com/video.mp4"}},
					}
					calls := 0
					original := http.DefaultClient
					http.DefaultClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
						calls++
						if r.URL.String() != base+"/v1/chat/completions" || r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer test-key" {
							t.Errorf("unexpected request: %s %s", r.Method, r.URL)
						}
						var body map[string]any
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							return nil, err
						}
						if body["model"] != selected || body["max_tokens"] != float64(1234) {
							t.Errorf("lost model/options: %v", body)
						}
						if test.wantEffort == "" {
							if _, ok := body["reasoning_effort"]; ok {
								t.Errorf("unexpected effort: %v", body)
							}
						} else if body["reasoning_effort"] != test.wantEffort {
							t.Errorf("effort: %v", body)
						}
						if test.thinking == "" {
							if _, ok := body["thinking"]; ok {
								t.Errorf("changed custom/legacy thinking: %v", body)
							}
						} else if !reflect.DeepEqual(body["thinking"], map[string]any{"type": test.thinking}) {
							t.Errorf("thinking: %v", body)
						}
						for _, field := range []string{"service_tier", "reasoning_split"} {
							if _, ok := body[field]; ok {
								t.Errorf("unexpected field %s", field)
							}
						}
						messages := body["messages"].([]any)
						if !reflect.DeepEqual(messages[1].(map[string]any)["content"], content) {
							t.Errorf("lost structured history: %v", messages)
						}
						if calls == 2 {
							if messages[len(messages)-1].(map[string]any)["tool_call_id"] != "call1" {
								t.Errorf("lost tool result: %v", messages)
							}
							if messages[len(messages)-2].(map[string]any)["reasoning_content"] != "first thought" {
								t.Errorf("lost reasoning continuity: %v", messages)
							}
						}
						answer := `{"choices":[{"message":{"content":"done"}}]}`
						if mode == "generate" && calls == 1 {
							answer = `{"choices":[{"message":{"content":"","reasoning_content":"first thought","tool_calls":[{"id":"call1","type":"function","function":{"name":"lookup","arguments":"{}"}}]}}]}`
						} else if mode != "generate" {
							if body["stream"] != true {
								t.Error("missing stream flag")
							}
							if mode == "token_callback" && calls == 1 {
								answer = "data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"first \"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"reasoning_content\":\"thought\",\"tool_calls\":[{\"index\":0,\"id\":\"call1\",\"type\":\"function\",\"function\":{\"name\":\"lookup\",\"arguments\":\"{}\"}}]},\"finish_reason\":\"tool_calls\"}]}\n\ndata: [DONE]\n\n"
							} else {
								answer = "data: {\"choices\":[{\"delta\":{\"content\":\"done\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"
							}
						}
						return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(answer))}, nil
					})}
					t.Cleanup(func() { http.DefaultClient = original })
					p := model.New("minimax", model.WithAPIKey("test-key"), model.WithBaseURL(endpoint), model.WithModel(test.selected), model.WithEffort(test.effort), model.WithThinking(test.mode), model.WithMaxTokens(1234), model.WithToolHandler(func(_ context.Context, call model.ToolCall) model.ToolResult {
						return model.ToolResult{ID: call.ID, Content: "result"}
					}))
					req := &model.Request{SystemPrompt: "system", Prompt: "describe", Messages: []model.Message{{Role: "user", Content: content}}, Tools: []model.Tool{{Name: "lookup"}}}
					if mode == "stream" {
						s, err := p.Stream(context.Background(), req)
						if err != nil {
							t.Fatal(err)
						}
						defer s.Close()
						chunk, err := s.Recv()
						if err != nil || chunk.Reply != "done" {
							t.Fatalf("chunk=%v error=%v", chunk, err)
						}
					} else {
						var options []model.GenerateOption
						var tokens string
						if mode == "token_callback" {
							options = append(options, model.WithTokenHandler(func(s string) { tokens += s }))
						}
						response, err := p.Generate(context.Background(), req, options...)
						if err != nil || response.Answer != "done" {
							t.Fatalf("response=%v error=%v", response, err)
						}
						if mode == "token_callback" && tokens != "done" {
							t.Errorf("tokens=%q", tokens)
						}
					}
					wantCalls := 2
					if mode == "stream" {
						wantCalls = 1
					}
					if calls != wantCalls {
						t.Errorf("requests=%d, want %d", calls, wantCalls)
					}
					if p.Options().Model != selected || p.Options().Effort != test.effort {
						t.Error("request changed saved options")
					}
				})
			}
		}
	}
}

func TestFlashRejectsUnsupportedOptionsBeforeHTTP(t *testing.T) {
	original := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Error("invalid options reached transport")
		return nil, fmt.Errorf("unexpected HTTP request")
	})}
	t.Cleanup(func() { http.DefaultClient = original })
	for _, option := range []model.Option{model.WithThinking(model.ThinkingDisabled), model.WithEffort("none"), model.WithEffort("invalid")} {
		p := NewProvider(model.WithAPIKey("test-key"), option)
		if _, err := p.Generate(context.Background(), &model.Request{Prompt: "hello"}); err == nil {
			t.Error("Generate accepted invalid options")
		}
		if _, err := p.Generate(context.Background(), &model.Request{Prompt: "hello"}, model.WithTokenHandler(func(string) {})); err == nil {
			t.Error("streaming Generate accepted invalid options")
		}
		if _, err := p.Stream(context.Background(), &model.Request{Prompt: "hello"}); err == nil {
			t.Error("Stream accepted invalid options")
		}
	}
}
