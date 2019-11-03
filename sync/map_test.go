package sync

import (
	"testing"
	"time"

	store "github.com/micro/go-micro/store"
	mem_store "github.com/micro/go-micro/store/memory"
	mem_lock "github.com/micro/go-micro/sync/lock/memory"
)

func TestIterate(t *testing.T) {
	s1 := mem_store.NewStore()
	s2 := mem_store.NewStore()
	recA := &store.Record{
		Key:   "A",
		Value: nil,
	}
	recB := &store.Record{
		Key:   "B",
		Value: nil,
	}
	s1.Write(recA)
	s1.Write(recB)
	s2.Write(recB)
	s2.Write(recA)

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
