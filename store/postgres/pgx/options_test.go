package pgx

import (
	"context"
	"testing"

	"go-micro.dev/v6/store"
)

func TestNewStoreContext(t *testing.T) {
	t.Run("defaults to background context", func(t *testing.T) {
		s := NewStore()

		if s.Options().Context == nil {
			t.Fatal("expected a non-nil default context")
		}
	})

	t.Run("preserves configured context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), struct{}{}, "value")
		s := NewStore(store.WithContext(ctx))

		if s.Options().Context != ctx {
			t.Fatal("expected the configured context to be preserved")
		}
	})

	t.Run("replaces configured nil context", func(t *testing.T) {
		s := NewStore(store.WithContext(nil))

		if s.Options().Context == nil {
			t.Fatal("expected a non-nil fallback context")
		}
	})
}
