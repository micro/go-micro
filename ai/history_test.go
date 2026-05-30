package ai

import (
	"context"
	"testing"
)

type mockModel struct {
	reply string
	calls int
}

func (m *mockModel) Init(...Option) error    { return nil }
func (m *mockModel) Options() Options        { return Options{} }
func (m *mockModel) String() string          { return "mock" }
func (m *mockModel) Stream(context.Context, *Request, ...GenerateOption) (Stream, error) {
	return nil, nil
}
func (m *mockModel) Generate(_ context.Context, req *Request, _ ...GenerateOption) (*Response, error) {
	m.calls++
	return &Response{Reply: m.reply}, nil
}

func TestHistory_AccumulatesMessages(t *testing.T) {
	m := &mockModel{reply: "hi"}
	hist := NewHistory(0)

	resp, err := Generate(context.Background(), m, &Request{
		Prompt:       "hello",
		SystemPrompt: "system",
		History:      hist,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Reply != "hi" {
		t.Errorf("reply = %q", resp.Reply)
	}
	if hist.Len() != 2 {
		t.Errorf("len = %d, want 2", hist.Len())
	}

	Generate(context.Background(), m, &Request{
		Prompt:  "again",
		History: hist,
	})
	if hist.Len() != 4 {
		t.Errorf("len = %d, want 4", hist.Len())
	}

	msgs := hist.Messages()
	if msgs[0].Role != "user" || msgs[0].Content != "hello" {
		t.Errorf("first = %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "hi" {
		t.Errorf("second = %+v", msgs[1])
	}
}

func TestHistory_Truncation(t *testing.T) {
	m := &mockModel{reply: "ok"}
	hist := NewHistory(4)

	for _, p := range []string{"one", "two", "three"} {
		Generate(context.Background(), m, &Request{Prompt: p, History: hist})
	}
	if hist.Len() != 4 {
		t.Errorf("len = %d, want 4", hist.Len())
	}
	msgs := hist.Messages()
	if msgs[0].Role != "user" || msgs[0].Content != "two" {
		t.Errorf("first retained = %+v", msgs[0])
	}
}

func TestHistory_Reset(t *testing.T) {
	m := &mockModel{reply: "ok"}
	hist := NewHistory(0)
	Generate(context.Background(), m, &Request{Prompt: "hello", History: hist})
	if hist.Len() == 0 {
		t.Fatal("expected messages")
	}
	hist.Reset()
	if hist.Len() != 0 {
		t.Errorf("len after reset = %d", hist.Len())
	}
}

func TestHistory_SnapshotIsCopy(t *testing.T) {
	m := &mockModel{reply: "ok"}
	hist := NewHistory(0)
	Generate(context.Background(), m, &Request{Prompt: "hello", History: hist})
	msgs := hist.Messages()
	msgs[0].Content = "mutated"
	if hist.Messages()[0].Content == "mutated" {
		t.Error("snapshot returned reference, not copy")
	}
}

func TestHistory_ToolCallsRecorded(t *testing.T) {
	tm := &toolModel{}
	hist := NewHistory(0)
	Generate(context.Background(), tm, &Request{Prompt: "do something", History: hist})
	if hist.Len() != 3 {
		t.Errorf("len = %d, want 3", hist.Len())
	}
}

func TestGenerate_WithoutHistory(t *testing.T) {
	m := &mockModel{reply: "ok"}
	resp, err := Generate(context.Background(), m, &Request{Prompt: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Reply != "ok" {
		t.Errorf("reply = %q", resp.Reply)
	}
}

type toolModel struct{}

func (m *toolModel) Init(...Option) error    { return nil }
func (m *toolModel) Options() Options        { return Options{} }
func (m *toolModel) String() string          { return "tool-mock" }
func (m *toolModel) Stream(context.Context, *Request, ...GenerateOption) (Stream, error) {
	return nil, nil
}
func (m *toolModel) Generate(_ context.Context, _ *Request, _ ...GenerateOption) (*Response, error) {
	return &Response{
		ToolCalls: []ToolCall{{ID: "1", Name: "test", Input: map[string]any{}}},
		Answer:    "done",
	}, nil
}
