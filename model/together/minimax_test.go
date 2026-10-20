package together

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
			fmt.Fprint(w, `[{"id":"MiniMaxAI/MiniMax-M2.7","type":"chat"},{"id":"MiniMaxAI/MiniMax-M3","type":"chat"},{"id":"MiniMaxAI/MiniMax-H3","type":"video"},{"id":"vendor/model","type":"chat"}]`)
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
		if body["model"] != "MiniMaxAI/MiniMax-M2.7" {
			t.Errorf("saved model changed: %v", body)
		}
		if body["stream"] == true {
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\ndata: [DONE]\n\n")
		} else {
			fmt.Fprint(w, `{"choices":[{"message":{"content":"ok"}}]}`)
		}
	}))
	defer server.Close()
	p := NewProvider(model.WithBaseURL(server.URL), model.WithAPIKey("test-key"), model.WithModel("MiniMaxAI/MiniMax-M2.7"))
	choices, err := model.ListModels(context.Background(), p)
	want := []string{"MiniMaxAI/MiniMax-M2.7", "MiniMaxAI/MiniMax-M3", "vendor/model"}
	if err != nil || !reflect.DeepEqual(choices, want) {
		t.Fatalf("choices=%v error=%v", choices, err)
	}
	raw, err := p.Models(context.Background())
	if err != nil || len(raw) != 3 {
		t.Fatalf("raw=%v error=%v", raw, err)
	}
	if p.Options().Model != "MiniMaxAI/MiniMax-M2.7" {
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
