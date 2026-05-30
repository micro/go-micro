package ai

import "testing"

func TestHistory_Add(t *testing.T) {
	h := NewHistory(0)
	h.Add("user", "hello")
	h.Add("assistant", "hi")

	if h.Len() != 2 {
		t.Errorf("len = %d, want 2", h.Len())
	}
	msgs := h.Messages()
	if msgs[0].Role != "user" || msgs[0].Content != "hello" {
		t.Errorf("first = %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "hi" {
		t.Errorf("second = %+v", msgs[1])
	}
}

func TestHistory_Truncation(t *testing.T) {
	h := NewHistory(3)
	for _, m := range []string{"a", "b", "c", "d", "e"} {
		h.Add("user", m)
	}
	if h.Len() != 3 {
		t.Errorf("len = %d, want 3", h.Len())
	}
	if h.Messages()[0].Content != "c" {
		t.Errorf("first retained = %+v", h.Messages()[0])
	}
}

func TestHistory_Reset(t *testing.T) {
	h := NewHistory(0)
	h.Add("user", "hello")
	h.Reset()
	if h.Len() != 0 {
		t.Errorf("len after reset = %d", h.Len())
	}
}

func TestHistory_SnapshotIsCopy(t *testing.T) {
	h := NewHistory(0)
	h.Add("user", "hello")
	msgs := h.Messages()
	msgs[0].Content = "mutated"
	if h.Messages()[0].Content == "mutated" {
		t.Error("snapshot returned reference, not copy")
	}
}

func TestHistory_Unlimited(t *testing.T) {
	h := NewHistory(0)
	for i := 0; i < 100; i++ {
		h.Add("user", "msg")
	}
	if h.Len() != 100 {
		t.Errorf("len = %d, want 100", h.Len())
	}
}
