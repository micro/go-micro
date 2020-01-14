package memory

import (
	"context"
	"fmt"
)

type flowStore struct {
	store map[string][]byte
}

// Create default in memory flow store
func DefaultFlowStore() *flowStore {
	return &flowStore{
		store: make(map[string][]byte),
	}

}

func (s *flowStore) Init() error {
	return nil
}

func (s *flowStore) Write(ctx context.Context, name string, data []byte) error {
	s.store[name] = data
	return nil
}

func (s *flowStore) Read(ctx context.Context, name string) ([]byte, error) {
	buf, ok := s.store[name]
	if !ok {
		return nil, fmt.Errorf("flow %s not found", name)
	}

	return buf, nil
}

func (s *flowStore) Close(ctx context.Context) error {
	return nil
}

func (s *flowStore) String() string {
	return "memory"
}
