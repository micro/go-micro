package memory

import (
	"context"
	"sync"
	"time"

	"github.com/micro/go-micro/flow"
)

type flowStore struct {
	sync.RWMutex
	store map[string][]*flow.Step
	stale map[string]int64
}

// Create default in memory flow store
func NewFlowStore() *flowStore {
	return &flowStore{
		store: make(map[string][]*flow.Step),
		stale: make(map[string]int64),
	}
}

func (s *flowStore) Init() error {
	return nil
}

func (s *flowStore) Modified(ctx context.Context, name string) int64 {
	s.RLock()
	stamp, ok := s.stale[name]
	s.RUnlock()
	if ok {
		return stamp
	}
	return 0
}

func (s *flowStore) Save(ctx context.Context, name string, steps []*flow.Step) error {
	s.Lock()
	s.store[name] = steps
	s.stale[name] = time.Now().Unix()
	s.Unlock()
	return nil
}

func (s *flowStore) Load(ctx context.Context, name string) ([]*flow.Step, error) {
	s.RLock()
	steps, ok := s.store[name]
	s.RUnlock()
	if !ok {
		return nil, flow.ErrFlowNotFound
	}

	return steps, nil
}

func (s *flowStore) Close(ctx context.Context) error {
	s.Lock()
	s.Unlock()
	return nil
}

func (s *flowStore) String() string {
	return "memory"
}
