package memory

import (
	"bytes"
	"context"
	"fmt"
)

type stateStore struct {
	store map[string]map[string][]byte
}

// Create default in memory state store
func DefaultStateStore() *stateStore {
	return &stateStore{
		store: make(map[string]map[string][]byte),
	}
}

func (s *stateStore) Init() error {
	return nil
}

// Update update a value (implement StateStore)
func (s *stateStore) Update(ctx context.Context, flow string, rid string, key []byte, oldval []byte, newval []byte) error {
	val, ok := s.store[fmt.Sprintf("%s-%s", flow, rid)][string(key)]
	if !ok {
		return fmt.Errorf("key not found %s", key)
	}
	if !bytes.Equal(val, oldval) {
		return fmt.Errorf("val in store not equal to provided")
	}

	s.store[fmt.Sprintf("%s-%s", flow, rid)][string(key)] = newval
	return nil
}

func (s *stateStore) Write(ctx context.Context, flow string, rid string, key []byte, val []byte) error {
	s.store[fmt.Sprintf("%s-%s", flow, rid)][string(key)] = val
	return nil
}

func (s *stateStore) Read(ctx context.Context, flow string, rid string, key []byte) ([]byte, error) {
	val, ok := s.store[fmt.Sprintf("%s-%s", flow, rid)][string(key)]
	if !ok {
		return nil, fmt.Errorf("key not found %s", key)
	}
	return val, nil
}

func (s *stateStore) Delete(ctx context.Context, flow string, rid string, key []byte) error {
	delete(s.store[fmt.Sprintf("%s-%s", flow, rid)], string(key))
	return nil
}

func (s *stateStore) String() string {
	return "memory"
}

func (s *stateStore) Clean(ctx context.Context, flow string, rid string) error {
	delete(s.store, fmt.Sprintf("%s-%s", flow, rid))
	return nil
}

func (s *stateStore) Close(ctx context.Context) error {
	return nil
}
