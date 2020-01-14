package memory

import (
	"context"
	"fmt"
)

type dataStore struct {
	store map[string]map[string][]byte
}

// Create default in memory data store
func DefaultDataStore() *dataStore {
	return &dataStore{
		store: make(map[string]map[string][]byte),
	}
}

func (s *dataStore) Init() error {
	return nil
}

func (s *dataStore) Write(ctx context.Context, flow string, rid string, key []byte, val []byte) error {
	s.store[fmt.Sprintf("%s-%s", flow, rid)][string(key)] = val
	return nil
}

func (s *dataStore) Read(ctx context.Context, flow string, rid string, key []byte) ([]byte, error) {
	val, ok := s.store[fmt.Sprintf("%s-%s", flow, rid)][string(key)]
	if !ok {
		return nil, fmt.Errorf("key not found %s", key)
	}
	return val, nil
}

func (s *dataStore) Delete(ctx context.Context, flow string, rid string, key []byte) error {
	delete(s.store[fmt.Sprintf("%s-%s", flow, rid)], string(key))
	return nil
}

func (s *dataStore) Clean(ctx context.Context, flow string, rid string) error {
	delete(s.store, fmt.Sprintf("%s-%s", flow, rid))
	return nil
}

func (s *dataStore) String() string {
	return "memory"
}

func (s *dataStore) Close(ctx context.Context) error {
	return nil
}
