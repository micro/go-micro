package ai

import "context"

// History accumulates conversation messages and feeds them into
// Generate calls. It handles the bookkeeping of adding user prompts,
// assistant replies, and tool call/result pairs so callers don't
// have to manage the message list manually.
//
// When the message count exceeds Limit, the oldest messages (after
// the system prompt) are dropped. Set Limit to 0 for unlimited.
type History struct {
	messages     []Message
	systemPrompt string
	limit        int
}

// NewHistory creates an empty History. limit controls the maximum
// number of messages retained (0 = unlimited). The system prompt is
// always preserved regardless of truncation.
func NewHistory(systemPrompt string, limit int) *History {
	return &History{
		systemPrompt: systemPrompt,
		limit:        limit,
	}
}

// Generate sends a user prompt through the model with the full
// conversation history. The user message, assistant reply, and any
// tool call/result pairs are appended to the history automatically.
func (h *History) Generate(ctx context.Context, m Model, prompt string, tools []Tool) (*Response, error) {
	h.add("user", prompt)

	resp, err := m.Generate(ctx, &Request{
		Prompt:       prompt,
		SystemPrompt: h.systemPrompt,
		Tools:        tools,
		Messages:     h.snapshot(),
	})
	if err != nil {
		return nil, err
	}

	// Record the assistant's reply.
	if resp.Reply != "" {
		h.add("assistant", resp.Reply)
	}

	// Record tool calls and results so subsequent turns have context.
	for _, tc := range resp.ToolCalls {
		h.add("assistant", tc)
	}
	if resp.Answer != "" {
		h.add("assistant", resp.Answer)
	}

	return resp, nil
}

// Messages returns a copy of the current message history.
func (h *History) Messages() []Message {
	return h.snapshot()
}

// Len returns the number of messages in the history.
func (h *History) Len() int {
	return len(h.messages)
}

// Reset clears all messages but keeps the system prompt and limit.
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
