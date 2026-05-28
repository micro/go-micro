package ai

import (
	"context"
	"testing"
)

type mockModel struct {
	reply string
	calls int
}

func (m *mockModel) Init(...Option) error { return nil }
func (m *mockModel) Options() Options     { return Options{} }
func (m *mockModel) String() string       { return "mock" }
func (m *mockModel) Stream(context.Context, *Request, ...GenerateOption) (Stream, error) {
	return nil, nil
}

func (m *mockModel) Generate(_ context.Context, req *Request, _ ...GenerateOption) (*Response, error) {
	m.calls++
	return &Response{Reply: m.reply}, nil
}

func TestHistory_AccumulatesMessages(t *testing.T) {
	m := &mockModel{reply: "hi"}
	h := NewHistory("system", 0)

	resp, err := h.Generate(context.Background(), m, "hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Reply != "hi" {
		t.Errorf("reply = %q", resp.Reply)
	}
	// user + assistant = 2
	if h.Len() != 2 {
		t.Errorf("len = %d, want 2", h.Len())
	}

	h.Generate(context.Background(), m, "again", nil)
	// 2 + user + assistant = 4
	if h.Len() != 4 {
		t.Errorf("len = %d, want 4", h.Len())
	}

	msgs := h.Messages()
	if msgs[0].Role != "user" || msgs[0].Content != "hello" {
		t.Errorf("first message = %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "hi" {
		t.Errorf("second message = %+v", msgs[1])
	}
	if msgs[2].Role != "user" || msgs[2].Content != "again" {
		t.Errorf("third message = %+v", msgs[2])
	}
}

func TestHistory_Truncation(t *testing.T) {
	m := &mockModel{reply: "ok"}
	h := NewHistory("system", 4)

	h.Generate(context.Background(), m, "one", nil)
	h.Generate(context.Background(), m, "two", nil)
	h.Generate(context.Background(), m, "three", nil)

	// 3 turns x 2 messages = 6, but limit is 4
	if h.Len() != 4 {
		t.Errorf("len = %d, want 4", h.Len())
	}

	msgs := h.Messages()
	// oldest messages should have been dropped
	if msgs[0].Role != "user" || msgs[0].Content != "two" {
		t.Errorf("first retained message = %+v, want user/two", msgs[0])
	}
}

func TestHistory_Reset(t *testing.T) {
	m := &mockModel{reply: "ok"}
	h := NewHistory("system", 0)

	h.Generate(context.Background(), m, "hello", nil)
	if h.Len() == 0 {
		t.Fatal("expected messages")
	}

	h.Reset()
	if h.Len() != 0 {
		t.Errorf("len after reset = %d", h.Len())
	}
}

func TestHistory_SnapshotIsCopy(t *testing.T) {
	m := &mockModel{reply: "ok"}
	h := NewHistory("system", 0)

	h.Generate(context.Background(), m, "hello", nil)
	msgs := h.Messages()
	msgs[0].Content = "mutated"

	if h.Messages()[0].Content == "mutated" {
		t.Error("Messages() returned a reference, not a copy")
	}
}

func TestHistory_ToolCallsRecorded(t *testing.T) {
	m := &mockModel{}
	m2 := &toolModel{}
	h := NewHistory("system", 0)

	h.Generate(context.Background(), m2, "do something", nil)

	// user + assistant (tool call) + assistant (answer) = 3
	if h.Len() != 3 {
		t.Errorf("len = %d, want 3", h.Len())
	}

	_ = m // avoid unused
}

type toolModel struct{}

func (m *toolModel) Init(...Option) error { return nil }
func (m *toolModel) Options() Options     { return Options{} }
func (m *toolModel) String() string       { return "tool-mock" }
func (m *toolModel) Stream(context.Context, *Request, ...GenerateOption) (Stream, error) {
	return nil, nil
}

func (m *toolModel) Generate(_ context.Context, _ *Request, _ ...GenerateOption) (*Response, error) {
	return &Response{
		ToolCalls: []ToolCall{{ID: "1", Name: "test", Input: map[string]any{}}},
		Answer:    "done",
	}, nil
}
