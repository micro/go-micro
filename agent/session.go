package agent

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"

	"go-micro.dev/v6/metadata"
	"go-micro.dev/v6/model"
	"go-micro.dev/v6/store"
)

// WithSession selects a conversation when calling an agent over RPC. The host
// must authorize access to the agent; a session ID is not an access credential.
func WithSession(ctx context.Context, id string) context.Context {
	return metadata.Set(ctx, "Micro-Session", id)
}

// RPC conversations are loaded from the existing store for each request. Keep
// requests serialized so two clients cannot overwrite the same history.
func (a *agentImpl) rpcSession(ctx context.Context) (*agentImpl, func(), error) {
	id, _ := metadata.Get(ctx, "Micro-Session")
	if id == "" {
		return a, func() {}, nil
	}
	if len(id) > 128 {
		return nil, nil, fmt.Errorf("agent: session ID exceeds 128 bytes")
	}
	release, err := a.lockSession(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	opts := a.opts
	opts.Session = id
	if opts.Memory != nil {
		memories, ok := opts.Memory.(SessionMemory)
		if !ok {
			release()
			return nil, nil, fmt.Errorf("agent: custom memory must implement SessionMemory for RPC sessions")
		}
		memory, err := memories.Session(id)
		if err != nil {
			release()
			return nil, nil, err
		}
		if memory == nil {
			release()
			return nil, nil, fmt.Errorf("agent: SessionMemory returned nil")
		}
		opts.Memory = memory
	}
	child := &agentImpl{opts: opts}
	if _, err := child.stateStore().List(store.ListLimit(1)); err != nil {
		release()
		return nil, nil, fmt.Errorf("agent session storage: %w", err)
	}
	return child, release, nil
}

// LoadHistory reads a stored conversation, including the default conversation
// when id is empty. It reports storage errors rather than treating them as an
// empty history. Custom memory implementations provide their own history access.
func LoadHistory(s store.Store, name, id string) ([]model.Message, error) {
	state, err := loadHistoryState(s, name, id)
	return state.Messages, err
}

func loadHistoryState(s store.Store, name, id string) (memoryState, error) {
	records, err := sessionStore(s, name, id).Read("history")
	if errors.Is(err, store.ErrNotFound) {
		return memoryState{}, nil
	}
	if err != nil {
		return memoryState{}, err
	}
	if len(records) == 0 {
		return memoryState{}, nil
	}
	var state memoryState
	if err := records[0].Decode(&state); err != nil {
		var legacy []model.Message
		if legacyErr := json.Unmarshal(records[0].Value, &legacy); legacyErr != nil {
			return memoryState{}, err
		}
		return memoryState{Messages: legacy}, nil
	}
	return state, nil
}

func sessionStore(s store.Store, name, id string) store.Store {
	if s == nil {
		s = store.DefaultStore
	}
	if id != "" {
		key, _ := json.Marshal([]string{name, id})
		return store.Scope(s, "agent_sessions", fmt.Sprintf("%x", sha256.Sum256(key)))
	}
	return store.Scope(s, "agent", name)
}

// SessionMemory lets a custom backend supply isolated conversation memory.
// Returned memories must retain state across calls and must not share messages
// between IDs. Hosts remain responsible for authorizing session access.
type SessionMemory interface {
	Memory
	Session(id string) (Memory, error)
}

type sessionLock struct {
	gate  chan struct{}
	users int
}

func (a *agentImpl) lockSession(ctx context.Context, id string) (func(), error) {
	a.sessionMu.Lock()
	if a.sessions == nil {
		a.sessions = make(map[string]*sessionLock)
	}
	lock := a.sessions[id]
	if lock == nil {
		lock = &sessionLock{gate: make(chan struct{}, 1)}
		a.sessions[id] = lock
	}
	lock.users++
	a.sessionMu.Unlock()
	drop := func() {
		a.sessionMu.Lock()
		lock.users--
		if lock.users == 0 {
			delete(a.sessions, id)
		}
		a.sessionMu.Unlock()
	}
	select {
	case lock.gate <- struct{}{}:
		return func() { <-lock.gate; drop() }, nil
	case <-ctx.Done():
		drop()
		return nil, ctx.Err()
	}
}
