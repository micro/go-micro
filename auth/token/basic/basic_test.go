package basic

import (
	"testing"
	"time"

	"github.com/micro/go-micro/v2/auth/token"
	"github.com/micro/go-micro/v2/store/memory"
)

func TestGenerate(t *testing.T) {
	store := memory.NewStore()
	b := NewTokenProvider(token.WithStore(store))

	_, err := b.Generate("test")
	if err != nil {
		t.Fatalf("Generate returned %v error, expected nil", err)
	}

	recs, err := store.List()
	if err != nil {
		t.Fatalf("Unable to read from store: %v", err)
	}
	if len(recs) != 1 {
		t.Errorf("Generate didn't write to the store, expected 1 record, got %v", len(recs))
	}
}

func TestInspect(t *testing.T) {
	store := memory.NewStore()
	b := NewTokenProvider(token.WithStore(store))

	t.Run("Valid token", func(t *testing.T) {
		md := map[string]string{"foo": "bar"}
		roles := []string{"admin"}
		subject := "test"

		opts := []token.GenerateOption{
			token.WithMetadata(md),
			token.WithRoles(roles...),
		}

		tok, err := b.Generate(subject, opts...)
		if err != nil {
			t.Fatalf("Generate returned %v error, expected nil", err)
		}

		tok2, err := b.Inspect(tok.Token)
		if err != nil {
			t.Fatalf("Inspect returned %v error, expected nil", err)
		}
		if tok2.Subject != subject {
			t.Errorf("Inspect returned %v as the token subject, expected %v", tok2.Subject, subject)
		}
		if len(tok2.Roles) != len(roles) {
			t.Errorf("Inspect returned %v roles, expected %v", len(tok2.Roles), len(roles))
		}
		if len(tok2.Metadata) != len(md) {
			t.Errorf("Inspect returned %v as the token metadata, expected %v", tok2.Metadata, md)
		}
	})

	t.Run("Expired token", func(t *testing.T) {
		tok, err := b.Generate("foo", token.WithExpiry(-10*time.Second))
		if err != nil {
			t.Fatalf("Generate returned %v error, expected nil", err)
		}

		if _, err = b.Inspect(tok.Token); err != token.ErrInvalidToken {
			t.Fatalf("Inspect returned %v error, expected %v", err, token.ErrInvalidToken)
		}
	})

	t.Run("Invalid token", func(t *testing.T) {
		_, err := b.Inspect("Invalid token")
		if err != token.ErrInvalidToken {
			t.Fatalf("Inspect returned %v error, expected %v", err, token.ErrInvalidToken)
		}
	})
}
