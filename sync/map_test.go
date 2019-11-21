package sync

import (
	"testing"
	"time"

	store "github.com/micro/go-micro/store"
	mocks "github.com/micro/go-micro/store/mocks"
	mem_lock "github.com/micro/go-micro/sync/lock/memory"
	"github.com/stretchr/testify/mock"
)

func TestIterate(t *testing.T) {
	recA := &store.Record{
		Key:   "A",
		Value: nil,
	}
	recB := &store.Record{
		Key:   "B",
		Value: nil,
	}
	s1 := &mocks.Store{}
	s2 := &mocks.Store{}
	s1.On("List").Return([]*store.Record{recA, recB}, nil)
	s2.On("List").Return([]*store.Record{recB, recA}, nil)
	s1.On("Write", mock.Anything).Return(nil)
	s2.On("Write", mock.Anything).Return(nil)

	f := func(key, val interface{}) error {
		time.Sleep(1 * time.Millisecond)
		return nil
	}
	l := mem_lock.NewLock()
	m1 := NewMap(WithStore(s1), WithLock(l))
	m2 := NewMap(WithStore(s2), WithLock(l))
	go func() {
		m2.Iterate(f)
	}()
	m1.Iterate(f)
}
