package typesafe

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-micro.dev/v6/model"
)

func TestDecide(t *testing.T) {
	var received model.DecisionRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/systemone" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing request headers")
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"jev-1.13.0","answers":{
			"urgent":{"type":"noul","noul":0},
			"route":{"type":"choice","choice":"support","probabilities":{"support":0.9,"sales":0.1},"confidence":0.8},
			"severity":{"type":"score","score":0.25,"probabilities":{"0":0.75,"1":0.25},"legend":{"0":"Low","1":"High"},"confidence":0.5}
		},"usage":{"input_tokens":100,"output_tokens":20}}`))
	}))
	defer server.Close()

	p := model.NewDecider("typesafe", model.WithAPIKey("test-key"), model.WithBaseURL(server.URL+"/"))
	if p == nil || p.String() != "typesafe" || p.Options().Model != "jev-latest" {
		t.Fatal("provider not registered with defaults")
	}
	req := &model.DecisionRequest{
		State: map[string]any{"message": "Please help"},
		Questions: map[string]model.Question{
			"urgent": {Type: model.Noul, Instructions: "Is this urgent?"},
			"route": {Type: model.Choice, Instructions: map[string]string{"question": "Who should handle this?"},
				Criteria: map[string]any{"support": "Product questions", "sales": nil}},
			"severity": {Type: model.Score, Instructions: "How severe?", Criteria: []string{"Low", "High"}},
		},
	}
	resp, err := p.Decide(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if received.Model != "jev-latest" || received.Questions["route"].Type != model.Choice || received.State.(map[string]any)["message"] != "Please help" {
		t.Fatalf("wrong request: %+v", received)
	}
	if req.Model != "" {
		t.Fatal("Decide mutated request")
	}
	if resp.Model != "jev-1.13.0" || resp.Answers["urgent"].Noul != 0 ||
		resp.Answers["route"].Choice != "support" || resp.Answers["route"].Confidence != 0.8 ||
		resp.Answers["severity"].Score != 0.25 || resp.Answers["severity"].Legend["1"] != "High" ||
		resp.Usage.TotalTokens != 120 {
		t.Fatalf("wrong response: %+v", resp)
	}
	req.Model = "jev-preview"
	if _, err := p.Decide(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if received.Model != "jev-preview" {
		t.Fatal("per-request model was ignored")
	}
	if err := p.Init(model.WithModel("configured-model")); err != nil {
		t.Fatal(err)
	}
	if p.Options().Model != "configured-model" || model.NewDecider("missing-provider") != nil {
		t.Fatal("incorrect provider options or lookup")
	}
}

func TestDecideErrors(t *testing.T) {
	for _, status := range []int{401, 422, 429, 529} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Retry-After", "2")
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"error":"request failed"}`))
			}))
			defer server.Close()
			p := NewProvider(model.WithAPIKey("test"), model.WithBaseURL(server.URL))
			_, err := p.Decide(context.Background(), noulRequest())
			var apiErr *model.HTTPError
			if !errors.As(err, &apiErr) || apiErr.StatusCode() != status {
				t.Fatalf("expected HTTP error %d, got %v", status, err)
			}
			if apiErr.RetryAfter() != 2*time.Second {
				t.Fatal("Retry-After was lost")
			}
			if calls != 1 {
				t.Fatalf("unexpected retry: %d calls", calls)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := NewProvider(model.WithAPIKey("test"), model.WithBaseURL("http://127.0.0.1:1"))
	if _, err := p.Decide(ctx, noulRequest()); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if _, err := p.Decide(ctx, nil); err == nil {
		t.Fatal("nil request accepted")
	}
	if _, err := NewProvider().Decide(ctx, noulRequest()); err == nil {
		t.Fatal("missing key accepted")
	}
}

func noulRequest() *model.DecisionRequest {
	return &model.DecisionRequest{State: "Help", Questions: map[string]model.Question{
		"urgent": {Type: model.Noul, Instructions: "Urgent?"},
	}}
}

func TestInvalidResponses(t *testing.T) {
	for _, body := range []string{
		`not json`,
		`{}`,
		`{"answers":{"urgent":{"type":"noul"}}}`,
		`{"answers":{"urgent":{"type":"noul","noul":null}}}`,
		`{"answers":{"urgent":{"type":"noul","noul":1.1}}}`,
		`{"answers":{"urgent":{"type":"choice","choice":"yes"}}}`,
		`{"answers":{"other":{"type":"noul","noul":0.5}}}`,
	} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(body))
			}))
			defer server.Close()
			p := NewProvider(model.WithAPIKey("test"), model.WithBaseURL(server.URL))
			if _, err := p.Decide(context.Background(), noulRequest()); err == nil {
				t.Fatal("malformed response accepted")
			}
		})
	}
}

func TestInvalidQuestions(t *testing.T) {
	for _, question := range []model.Question{
		{Type: "unknown", Instructions: "Question"},
		{Type: model.Noul},
		{Type: model.Noul, Instructions: true},
		{Type: model.Noul, Instructions: "Question", Criteria: map[string]string{"maybe": "Maybe"}},
		{Type: model.Choice, Instructions: "Question", Criteria: []string{"a", "b"}},
		{Type: model.Choice, Instructions: "Question", Criteria: map[string]string{}},
		{Type: model.Score, Instructions: "Question", Criteria: []string{"only"}},
	} {
		req := noulRequest()
		req.Questions["urgent"] = question
		p := NewProvider(model.WithAPIKey("test"), model.WithBaseURL("http://127.0.0.1:1"))
		_, err := p.Decide(context.Background(), req)
		if err == nil || !strings.Contains(err.Error(), "typesafe: question") {
			t.Fatalf("expected local validation error for %+v, got %v", question, err)
		}
	}
}

func TestInvalidChoiceAndScore(t *testing.T) {
	tests := []struct {
		question model.Question
		answer   string
	}{
		{model.Question{Type: model.Choice, Criteria: map[string]string{"a": "A", "b": "B"}}, `{"type":"choice","choice":"c","probabilities":{"a":1,"b":0},"confidence":1}`},
		{model.Question{Type: model.Choice, Criteria: map[string]string{"a": "A", "b": "B"}}, `{"type":"choice","choice":"a","probabilities":{"a":0.2,"b":0.2},"confidence":0}`},
		{model.Question{Type: model.Choice, Criteria: map[string]string{"a": "A", "b": "B"}}, `{"type":"choice","choice":"a","probabilities":{"a":1,"c":0},"confidence":1}`},
		{model.Question{Type: model.Score, Criteria: []string{"Low", "High"}}, `{"type":"score","score":2,"probabilities":{"0":0,"1":1},"legend":{"0":"Low","1":"High"},"confidence":1}`},
		{model.Question{Type: model.Score, Criteria: []string{"Low", "High"}}, `{"type":"score","score":0,"probabilities":{"0":1,"1":0},"legend":{"0":"Low","1":"High"}}`},
		{model.Question{Type: model.Choice, Criteria: map[string]string{"a": "A", "b": "B"}}, `{"type":"choice","choice":"a","probabilities":{"a":1,"b":null},"confidence":1}`},
	}
	for _, tt := range tests {
		_, err := decodeResponse([]byte(`{"answers":{"q":`+tt.answer+`}}`), map[string]model.Question{"q": tt.question})
		if err == nil {
			t.Errorf("invalid answer accepted: %s", tt.answer)
		}
	}
}
