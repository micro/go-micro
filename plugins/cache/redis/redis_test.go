package redis

import (
	"context"
	"os"
	"testing"
	"time"

	"go-micro.dev/v4/cache"
)

var (
	ctx              = context.TODO()
	key  string      = "redistestkey"
	val  interface{} = "hello go-micro"
	addr             = cache.WithAddress("redis://127.0.0.1:6379")
)

// TestMemCache tests the in-memory cache implementation.
func TestCache(t *testing.T) {
	if len(os.Getenv("LOCAL")) == 0 {
		t.Skip()
	}

	t.Run("CacheGetMiss", func(t *testing.T) {
		if _, _, err := NewCache(addr).Get(ctx, key); err == nil {
			t.Error("expected to get no value from cache")
		}
	})

	t.Run("CacheGetHit", func(t *testing.T) {
		c := NewCache(addr)

		if err := c.Put(ctx, key, val, 0); err != nil {
			t.Error(err)
		}

		if a, _, err := c.Get(ctx, key); err != nil {
			t.Errorf("Expected a value, got err: %s", err)
		} else if string(a.([]byte)) != val {
			t.Errorf("Expected '%v', got '%v'", val, a)
		}
	})

	t.Run("CacheGetExpired", func(t *testing.T) {
		c := NewCache(addr)
		d := 20 * time.Millisecond

		if err := c.Put(ctx, key, val, d); err != nil {
			t.Error(err)
		}

		<-time.After(25 * time.Millisecond)
		if _, _, err := c.Get(ctx, key); err == nil {
			t.Error("expected to get no value from cache")
		}
	})

	t.Run("CacheGetValid", func(t *testing.T) {
		c := NewCache(addr)
		e := 25 * time.Millisecond

		if err := c.Put(ctx, key, val, e); err != nil {
			t.Error(err)
		}

		<-time.After(20 * time.Millisecond)
		if _, _, err := c.Get(ctx, key); err != nil {
			t.Errorf("expected a value, got err: %s", err)
		}
	})

	t.Run("CacheDeleteHit", func(t *testing.T) {
		c := NewCache(addr)

		if err := c.Put(ctx, key, val, 0); err != nil {
			t.Error(err)
		}

		if err := c.Delete(ctx, key); err != nil {
			t.Errorf("Expected to delete an item, got err: %s", err)
		}

		if _, _, err := c.Get(ctx, key); err == nil {
			t.Errorf("Expected error")
		}
	})
}
