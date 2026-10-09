package agent

import (
	"errors"
	"fmt"
	"strings"

	"go-micro.dev/v6/model"
	"go-micro.dev/v6/store"
)

// MemoryCompactor is an optional explicit compaction capability. Compaction
// retains recent messages and archives older messages for retrieval.
type MemoryCompactor interface{ Compact(keepRecent int) error }

func (m *storeMemory) Compact(keepRecent int) error {
	if keepRecent < 1 {
		return errors.New("agent: keep at least one recent message")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	messages := m.hist.Messages()
	if len(messages) <= keepRecent {
		return nil
	}
	cut := len(messages) - keepRecent
	summarize := m.compaction.Summarize
	if summarize == nil {
		summarize = defaultMemorySummary
	}
	summary := summarize(messages[:cut])
	if summary.Role == "" {
		summary.Role = "system"
	}
	archive := append([]model.Message(nil), m.archive...)
	if !m.retrieveAll {
		archive = append(archive, messages[:cut]...)
	}
	active := append([]model.Message{summary}, messages[cut:]...)
	state := memoryState{Messages: active, Archive: archive, Summary: fmt.Sprint(summary.Content)}
	if m.store != nil && m.key != "" {
		if err := m.store.Write(store.NewRecord(m.key, state)); err != nil {
			return err
		}
	}
	m.archive, m.summary = archive, state.Summary
	m.hist.Reset()
	for _, message := range active {
		m.hist.Add(message.Role, message.Content)
	}
	return nil
}

// CompactHistory compacts a paused agent's conversation using its memory backend.
func CompactHistory(a Agent, keepRecent int) error {
	compactor, ok := a.(interface{ CompactHistory(int) error })
	if !ok {
		return errors.New("agent: history compaction unsupported")
	}
	return compactor.CompactHistory(keepRecent)
}
func (a *agentImpl) CompactHistory(keepRecent int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.mem == nil {
		a.setup()
	}
	compactor, ok := a.mem.(MemoryCompactor)
	if !ok {
		return errors.New("agent: memory does not support compaction")
	}
	return compactor.Compact(keepRecent)
}

// SearchHistory searches retained and archived messages in a stored conversation.
// It performs case-insensitive substring matching, newest first.
func SearchHistory(s store.Store, name, id, query string, limit int) ([]model.Message, error) {
	state, err := loadHistoryState(s, name, id)
	if err != nil {
		return nil, err
	}
	return searchMessages(append(state.Archive, state.Messages...), query, limit), nil
}
func searchMessages(messages []model.Message, query string, limit int) []model.Message {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}
	if limit <= 0 {
		limit = 20
	}
	var result []model.Message
	seen := map[string]bool{}
	for i := len(messages) - 1; i >= 0 && len(result) < limit; i-- {
		message := messages[i]
		content := fmt.Sprint(message.Content)
		key := message.Role + "\x00" + content
		if !seen[key] && strings.Contains(strings.ToLower(content), query) {
			result = append(result, message)
			seen[key] = true
		}
	}
	return result
}
