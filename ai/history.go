package ai

// History is a convenience for accumulating conversation messages
// with automatic truncation. Use it to build Request.Messages for
// multi-turn conversations.
//
//	hist := ai.NewHistory(50)
//	hist.Add("user", "hello")
//	resp, _ := m.Generate(ctx, &ai.Request{Messages: hist.Messages(), Prompt: "next"})
//	hist.Add("assistant", resp.Reply)
type History struct {
	messages []Message
	limit    int
}

// NewHistory creates an empty History. limit controls the maximum
// number of messages retained (0 = unlimited).
func NewHistory(limit int) *History {
	return &History{limit: limit}
}

// Add appends a message and truncates if over limit.
func (h *History) Add(role string, content any) {
	h.messages = append(h.messages, Message{Role: role, Content: content})
	if h.limit > 0 && len(h.messages) > h.limit {
		h.messages = h.messages[len(h.messages)-h.limit:]
	}
}

// Messages returns a copy of the accumulated messages.
func (h *History) Messages() []Message {
	out := make([]Message, len(h.messages))
	copy(out, h.messages)
	return out
}

// Len returns the number of messages.
func (h *History) Len() int {
	return len(h.messages)
}

// Reset clears all messages.
func (h *History) Reset() {
	h.messages = nil
}
