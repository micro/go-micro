package token

import (
	"os"
	"testing"
	"time"

	"go-micro.dev/v5/auth"
)

func TestGenerate(t *testing.T) {
	privKey, err := os.ReadFile("test/sample_key")
	if err != nil {
		t.Fatalf("Unable to read private key: %v", err)
	}

	j := New(
		WithPrivateKey(string(privKey)),
	)

	_, err = j.Generate(&auth.Account{ID: "test"})
	if err != nil {
		t.Fatalf("Generate returned %v error, expected nil", err)
	}
}

func TestInspect(t *testing.T) {
	pubKey, err := os.ReadFile("test/sample_key.pub")
	if err != nil {
		t.Fatalf("Unable to read public key: %v", err)
	}
	privKey, err := os.ReadFile("test/sample_key")
	if err != nil {
		t.Fatalf("Unable to read private key: %v", err)
	}

	j := New(
		WithPublicKey(string(pubKey)),
		WithPrivateKey(string(privKey)),
	)

	t.Run("Valid token", func(t *testing.T) {
		md := map[string]string{"foo": "bar"}
		scopes := []string{"admin"}
		subject := "test"

		acc := &auth.Account{ID: subject, Scopes: scopes, Metadata: md}
		tok, err := j.Generate(acc)
		if err != nil {
			t.Fatalf("Generate returned %v error, expected nil", err)
		}

		tok2, err := j.Inspect(tok.Token)
		if err != nil {
			t.Fatalf("Inspect returned %v error, expected nil", err)
		}
		if acc.ID != subject {
			t.Errorf("Inspect returned %v as the token subject, expected %v", acc.ID, subject)
		}
		if len(tok2.Scopes) != len(scopes) {
			t.Errorf("Inspect returned %v scopes, expected %v", len(tok2.Scopes), len(scopes))
		}
		if len(tok2.Metadata) != len(md) {
			t.Errorf("Inspect returned %v as the token metadata, expected %v", tok2.Metadata, md)
		}
	})

	t.Run("Expired token", func(t *testing.T) {
		tok, err := j.Generate(&auth.Account{}, WithExpiry(-10*time.Second))
		if err != nil {
			t.Fatalf("Generate returned %v error, expected nil", err)
		}

		if _, err = j.Inspect(tok.Token); err != ErrInvalidToken {
			t.Fatalf("Inspect returned %v error, expected %v", err, ErrInvalidToken)
		}
	})

	t.Run("Invalid token", func(t *testing.T) {
		_, err := j.Inspect("Invalid token")
		if err != ErrInvalidToken {
			t.Fatalf("Inspect returned %v error, expected %v", err, ErrInvalidToken)
		}
	})
}
