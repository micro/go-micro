package memory

import (
	"bytes"
	"context"
	"fmt"
	"sync"
)

type dataStore struct {
	mu    sync.RWMutex
	store map[string]map[string][]byte
}

// Create default in memory state store
func NewDataStore() *dataStore {
	return &dataStore{
		store: make(map[string]map[string][]byte),
	}
}

func (s *dataStore) Init() error {
	return nil
}

// Update update a value (implement StateStore)
func (s *dataStore) Update(ctx context.Context, flow string, rid string, key []byte, oldval []byte, newval []byte) error {
	s.mu.RLock()
	val, ok := s.store[fmt.Sprintf("%s-%s", flow, rid)][string(key)]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("key not found %s", key)
	}
	if !bytes.Equal(val, oldval) {
		return fmt.Errorf("val in store not equal to provided")
	}

	s.mu.Lock()
	s.store[fmt.Sprintf("%s-%s", flow, rid)][string(key)] = newval
	s.mu.Unlock()
	return nil
}

func (s *dataStore) Write(ctx context.Context, flow string, rid string, key []byte, val []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.store[fmt.Sprintf("%s-%s", flow, rid)]
	if !ok {
		s.store[fmt.Sprintf("%s-%s", flow, rid)] = make(map[string][]byte)
	}
	s.store[fmt.Sprintf("%s-%s", flow, rid)][string(key)] = val
	return nil
}

func (s *dataStore) Read(ctx context.Context, flow string, rid string, key []byte) ([]byte, error) {
	s.mu.RLock()
	val, ok := s.store[fmt.Sprintf("%s-%s", flow, rid)][string(key)]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("key not found %s", key)
	}
	return val, nil
}

func (s *dataStore) Delete(ctx context.Context, flow string, rid string, key []byte) error {
	s.mu.Lock()
	delete(s.store[fmt.Sprintf("%s-%s", flow, rid)], string(key))
	s.mu.Unlock()
	return nil
}

func (s *dataStore) String() string {
	return "memory"
}

func (s *dataStore) Clean(ctx context.Context, flow string, rid string) error {
	s.mu.Lock()
	delete(s.store, fmt.Sprintf("%s-%s", flow, rid))
	s.mu.Unlock()
	return nil
}

func (s *dataStore) Close(ctx context.Context) error {
	return nil
}
