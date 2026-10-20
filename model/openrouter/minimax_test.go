package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"go-micro.dev/v6/model"
)

func TestMiniMaxChoicesPreserveExplicitModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			fmt.Fprint(w, `{"data":[{"id":"vendor/model"},{"id":"minimax/minimax-m2.7"},{"id":"minimax/minimax-m3"},{"id":"minimax/minimax-m3.1-flash:free"},{"id":"minimax/minimax-m3.1-flash"},{"id":"minimax/hailuo"},{"id":"minimax/speech-02"},{"id":"minimax/minimax-video-01"},{"id":"minimax/minimax-01"}]}`)
			return
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path=%s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if body["model"] != "minimax/minimax-m2.7" {
			t.Errorf("saved model changed: %v", body)
		}
		if body["stream"] == true {
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\ndata: [DONE]\n\n")
		} else {
			fmt.Fprint(w, `{"choices":[{"message":{"content":"ok"}}]}`)
		}
	}))
	defer server.Close()
	p := NewProvider(model.WithBaseURL(server.URL), model.WithAPIKey("test-key"), model.WithModel("minimax/minimax-m2.7"))
	choices, err := model.ListModels(context.Background(), p)
	want := []string{"minimax/hailuo", "minimax/minimax-m3.1-flash:free", "minimax/minimax-m3", "minimax/minimax-video-01", "minimax/speech-02", "vendor/model"}
	if err != nil || !reflect.DeepEqual(choices, want) {
		t.Fatalf("choices=%v error=%v", choices, err)
	}
	// The raw upstream catalog remains available to callers that need it.
	raw, err := p.Models(context.Background())
	if err != nil || len(raw) != 9 {
		t.Fatalf("raw catalog=%v error=%v", raw, err)
	}
	if p.Options().Model != "minimax/minimax-m2.7" {
		t.Fatal("listing reset saved model")
	}
	if _, err := p.Generate(context.Background(), &model.Request{Prompt: "hi"}); err != nil {
		t.Fatal(err)
	}
	s, err := p.Stream(context.Background(), &model.Request{Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.Recv(); err != nil {
		t.Fatal(err)
	}
}

func TestMiniMaxGatewayRequest(t *testing.T) {
	for _, selected := range []string{"minimax/minimax-m3.1-flash:free", "minimax/minimax-m3"} {
		for _, mode := range []string{"generate", "stream", "token_callback"} {
			t.Run(selected+"/"+mode, func(t *testing.T) {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					if r.URL.Path != "/v1/chat/completions" {
						t.Errorf("path=%s", r.URL.Path)
					}
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
						return
					}
					if body["model"] != selected || !reflect.DeepEqual(body["reasoning"], map[string]any{"effort": "high"}) {
						t.Errorf("gateway request=%v", body)
					}
					for _, field := range []string{"reasoning_effort", "thinking", "service_tier"} {
						if _, ok := body[field]; ok {
							t.Errorf("unexpected direct field %s", field)
						}
					}
					if mode == "stream" {
						fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\ndata: [DONE]\n\n")
					} else {
						fmt.Fprint(w, `{"choices":[{"message":{"content":"ok"}}]}`)
					}
				}))
				defer server.Close()
				p := NewProvider(model.WithBaseURL(server.URL), model.WithAPIKey("test-key"), model.WithModel(selected), model.WithEffort("high"))
				if mode == "stream" {
					s, err := p.Stream(context.Background(), &model.Request{Prompt: "hi"})
					if err != nil {
						t.Fatal(err)
					}
					defer s.Close()
					if _, err := s.Recv(); err != nil {
						t.Fatal(err)
					}
				} else {
					var options []model.GenerateOption
					var tokens string
					if mode == "token_callback" {
						options = append(options, model.WithTokenHandler(func(s string) { tokens += s }))
					}
					if _, err := p.Generate(context.Background(), &model.Request{Prompt: "hi"}, options...); err != nil {
						t.Fatal(err)
					}
					if mode == "token_callback" && tokens != "ok" {
						t.Errorf("tokens=%q", tokens)
					}
				}
				if calls != 1 {
					t.Errorf("requests=%d", calls)
				}
			})
		}
	}
}

func TestMiniMaxChoicesDoNotInventRoutes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[{"id":"minimax/minimax-m2.7"},{"id":"minimax/minimax-m3"},{"id":"vendor/model"}]}`)
	}))
	defer server.Close()
	p := NewProvider(model.WithBaseURL(server.URL))
	choices, err := model.ListModels(context.Background(), p)
	if err != nil || !reflect.DeepEqual(choices, []string{"minimax/minimax-m3", "vendor/model"}) {
		t.Fatalf("choices=%v error=%v", choices, err)
	}
}
