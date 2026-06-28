package agent

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"go-micro.dev/v6/ai"
	"go-micro.dev/v6/store"
)

// Memory is an agent's conversation memory. Like the rest of the
// framework it is pluggable: the default is store-backed and durable
// across restarts, but any implementation can be supplied with
// WithMemory — in-process, a database, or a semantic/vector store.
type Memory interface {
	// Add appends a message to the conversation.
	Add(role, content string)
	// Messages returns the retained conversation, oldest first.
	Messages() []ai.Message
	// Clear resets the conversation.
	Clear()
}

// MemoryCompaction configures deterministic, store-backed context compaction
// for the default memory implementation. When the retained conversation grows
// past MaxMessages, older turns are collapsed into a summary message while the
// newest KeepRecent turns stay verbatim for provider-neutral continuity.
type MemoryCompaction struct {
	MaxMessages int
	KeepRecent  int
}

// MemoryRecall is implemented by memory backends that can retrieve durable
// prior context relevant to a new turn without replaying every stored message.
type MemoryRecall interface {
	Recall(query string, limit int) []ai.Message
}

// NewMemory returns the default store-backed memory: an in-process
// conversation buffer (truncated to limit) that persists to the store
// under key, so an agent picks up where it left off after a restart.
// A nil store or empty key yields non-persistent memory.
func NewMemory(s store.Store, key string, limit int) Memory {
	m := &storeMemory{store: s, key: key, hist: ai.NewHistory(limit)}
	m.load()
	return m
}

// NewCompactingMemory returns store-backed memory with explicit compaction and
// retrieval controls. It keeps all messages in the backing store, compacts older
// turns into a deterministic summary when the conversation exceeds maxMessages,
// and lets callers recall relevant prior turns with Recall.
func NewCompactingMemory(s store.Store, key string, maxMessages, keepRecent int) Memory {
	if keepRecent <= 0 {
		keepRecent = maxMessages / 2
	}
	if keepRecent < 1 {
		keepRecent = 1
	}
	m := &storeMemory{
		store: s,
		key:   key,
		// Use an unlimited buffer here; compaction, not truncation, decides
		// what remains in active context so a summary can preserve older turns.
		hist: ai.NewHistory(0),
		compaction: MemoryCompaction{
			MaxMessages: maxMessages,
			KeepRecent:  keepRecent,
		},
	}
	m.load()
	m.compact()
	return m
}

// NewInMemory returns conversation memory that is not persisted.
func NewInMemory(limit int) Memory {
	return &storeMemory{hist: ai.NewHistory(limit)}
}

// storeMemory is the default Memory: an ai.History buffer optionally
// persisted to a store.
type storeMemory struct {
	mu         sync.Mutex
	store      store.Store
	key        string
	hist       *ai.History
	compaction MemoryCompaction
	archive    []ai.Message
}

func (m *storeMemory) Add(role, content string) {
	m.mu.Lock()
	m.hist.Add(role, content)
	m.mu.Unlock()
	m.compact()
	m.save()
}

func (m *storeMemory) Messages() []ai.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.hist.Messages()
}

func (m *storeMemory) Clear() {
	m.mu.Lock()
	m.hist.Reset()
	m.archive = nil
	m.mu.Unlock()
	m.save()
}

// Recall returns archived messages whose content contains words from query.
// It is deterministic and provider-neutral: no embeddings or model calls are
// required, but semantic/vector stores can replace Memory for richer retrieval.
func (m *storeMemory) Recall(query string, limit int) []ai.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 {
		limit = 5
	}
	terms := recallTerms(query)
	type match struct {
		msg   ai.Message
		score int
		index int
	}
	matches := make([]match, 0, len(m.archive))
	for i := len(m.archive) - 1; i >= 0; i-- {
		msg := m.archive[i]
		if score := recallScore(msg, terms); score > 0 {
			matches = append(matches, match{msg: msg, score: score, index: i})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].index > matches[j].index
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}
	out := make([]ai.Message, 0, len(matches))
	for _, match := range matches {
		out = append(out, match.msg)
	}
	return out
}

func (m *storeMemory) load() {
	if m.store == nil || m.key == "" {
		return
	}
	recs, err := m.store.Read(m.key)
	if err != nil || len(recs) == 0 {
		return
	}
	var state memoryState
	if err := json.Unmarshal(recs[0].Value, &state); err != nil {
		var msgs []ai.Message
		if err := json.Unmarshal(recs[0].Value, &msgs); err != nil {
			return
		}
		state.Messages = msgs
	}
	m.mu.Lock()
	m.archive = state.Archive
	for _, msg := range state.Messages {
		m.hist.Add(msg.Role, msg.Content)
	}
	m.mu.Unlock()
}

func (m *storeMemory) save() {
	if m.store == nil || m.key == "" {
		return
	}
	m.mu.Lock()
	data, err := json.Marshal(memoryState{
		Messages: m.hist.Messages(),
		Archive:  m.archive,
	})
	m.mu.Unlock()
	if err != nil {
		return
	}
	_ = m.store.Write(&store.Record{Key: m.key, Value: data})
}

func (m *storeMemory) compact() {
	if m.compaction.MaxMessages <= 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	msgs := m.hist.Messages()
	if len(msgs) <= m.compaction.MaxMessages {
		return
	}
	keep := m.compaction.KeepRecent
	if keep <= 0 || keep >= m.compaction.MaxMessages {
		keep = m.compaction.MaxMessages - 1
	}
	if keep < 1 {
		keep = 1
	}
	cut := len(msgs) - keep
	older := msgs[:cut]
	recent := msgs[cut:]
	m.archive = append(m.archive, older...)
	summary := ai.Message{
		Role:    "system",
		Content: fmt.Sprintf("Conversation memory summary: %s", summarizeMessages(older)),
	}
	m.hist.Reset()
	m.hist.Add(summary.Role, summary.Content)
	for _, msg := range recent {
		m.hist.Add(msg.Role, msg.Content)
	}
}

func summarizeMessages(msgs []ai.Message) string {
	var b strings.Builder
	for i, msg := range msgs {
		if i > 0 {
			b.WriteString(" | ")
		}
		fmt.Fprintf(&b, "%s: %s", msg.Role, compactText(fmt.Sprint(msg.Content), 120))
	}
	return b.String()
}

func compactText(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if max > 0 && len(s) > max {
		return s[:max] + "…"
	}
	return s
}

func recallScore(msg ai.Message, terms []string) int {
	text := strings.ToLower(fmt.Sprint(msg.Content))
	score := 0
	for _, term := range terms {
		if strings.Contains(text, term) {
			score++
		}
	}
	return score
}

func recallTerms(query string) []string {
	seen := map[string]bool{}
	var terms []string
	for _, term := range strings.Fields(strings.ToLower(query)) {
		term = strings.Trim(term, ".,!?;:\"'()[]{}")
		if len(term) < 3 || seen[term] {
			continue
		}
		seen[term] = true
		terms = append(terms, term)
	}
	return terms
}

type memoryState struct {
	Messages []ai.Message `json:"messages"`
	Archive  []ai.Message `json:"archive,omitempty"`
}
