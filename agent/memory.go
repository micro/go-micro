package agent

import (
	"encoding/json"
	"sync"

	"go-micro.dev/v5/ai"
	"go-micro.dev/v5/store"
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

// NewMemory returns the default store-backed memory: an in-process
// conversation buffer (truncated to limit) that persists to the store
// under key, so an agent picks up where it left off after a restart.
// A nil store or empty key yields non-persistent memory.
func NewMemory(s store.Store, key string, limit int) Memory {
	m := &storeMemory{store: s, key: key, hist: ai.NewHistory(limit)}
	m.load()
	return m
}

// NewInMemory returns conversation memory that is not persisted.
func NewInMemory(limit int) Memory {
	return &storeMemory{hist: ai.NewHistory(limit)}
}

// storeMemory is the default Memory: an ai.History buffer optionally
// persisted to a store.
type storeMemory struct {
	mu    sync.Mutex
	store store.Store
	key   string
	hist  *ai.History
}

func (m *storeMemory) Add(role, content string) {
	m.mu.Lock()
	m.hist.Add(role, content)
	m.mu.Unlock()
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
	m.mu.Unlock()
	m.save()
}

func (m *storeMemory) load() {
	if m.store == nil || m.key == "" {
		return
	}
	recs, err := m.store.Read(m.key)
	if err != nil || len(recs) == 0 {
		return
	}
	var msgs []ai.Message
	if err := json.Unmarshal(recs[0].Value, &msgs); err != nil {
		return
	}
	m.mu.Lock()
	for _, msg := range msgs {
		m.hist.Add(msg.Role, msg.Content)
	}
	m.mu.Unlock()
}

func (m *storeMemory) save() {
	if m.store == nil || m.key == "" {
		return
	}
	m.mu.Lock()
	data, err := json.Marshal(m.hist.Messages())
	m.mu.Unlock()
	if err != nil {
		return
	}
	m.store.Write(&store.Record{Key: m.key, Value: data})
}
