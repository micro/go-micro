package ai

// History accumulates conversation messages for multi-turn use.
// Pass it via Request.History and the Generate function handles
// the bookkeeping automatically.
//
// When the message count exceeds the limit, the oldest messages
// are dropped. Set limit to 0 for unlimited.
type History struct {
	messages []Message
	limit    int
}

// NewHistory creates an empty History. limit controls the maximum
// number of messages retained (0 = unlimited).
func NewHistory(limit int) *History {
	return &History{limit: limit}
}

// Messages returns a copy of the current message history.
func (h *History) Messages() []Message {
	return h.snapshot()
}

// Len returns the number of messages in the history.
func (h *History) Len() int {
	return len(h.messages)
}

// Reset clears all messages.
func (h *History) Reset() {
	h.messages = nil
}

func (h *History) add(role string, content any) {
	h.messages = append(h.messages, Message{Role: role, Content: content})
	h.truncate()
}

func (h *History) truncate() {
	if h.limit <= 0 || len(h.messages) <= h.limit {
		return
	}
	drop := len(h.messages) - h.limit
	h.messages = h.messages[drop:]
}

func (h *History) snapshot() []Message {
	out := make([]Message, len(h.messages))
	copy(out, h.messages)
	return out
}
