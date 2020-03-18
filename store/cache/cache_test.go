package cache

import (
	"context"
	"testing"
	"time"

	"github.com/micro/go-micro/v2/store"
	"github.com/micro/go-micro/v2/store/memory"
)

func TestCacheTicker(t *testing.T) {
	l0 := memory.NewStore()
	l0.Init()
	l1 := memory.NewStore()
	l1.Init()
	l2 := memory.NewStore()
	l2.Init()
	c := NewCache(Stores(l0, l1, l2), SyncInterval(1*time.Second), SyncMultiplier(2))

	if err := c.Init(store.WithContext(context.Background())); err != nil {
		t.Fatal(err)
	}

	time.Sleep(30 * time.Second)
}
