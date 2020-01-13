package flow

import (
	"context"
	"fmt"
)

/*
type FlowStore interface {
	Init() error
	Save(flow string, services []*FlowOperation) error
	Load(flow string) ([]*FlowOperation, error)
	Append(flow string, service *FlowOperation) error
	Delete(flow string, service *FlowOperation) error
}
*/

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

func (s *flowStore) Write(ctx context.Context, flow string, data []byte) error {
	s.store[flow] = data
	return nil
}

func (s *flowStore) Read(ctx context.Context, flow string) ([]byte, error) {
	buf, ok := s.store[flow]
	if !ok {
		return nil, fmt.Errorf("flow %s not found", flow)
	}

	return buf, nil
}

func (s *flowStore) Close(ctx context.Context) error {
	return nil
}

func (s *flowStore) String() string {
	return "memory"
}
