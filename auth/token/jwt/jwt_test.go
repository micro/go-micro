package jwt

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/micro/go-micro/v2/auth/token"
)

func TestGenerate(t *testing.T) {
	privKey, err := ioutil.ReadFile("test/sample_key")
	if err != nil {
		t.Fatalf("Unable to read private key: %v", err)
	}

	j := NewTokenProvider(
		token.WithPrivateKey(string(privKey)),
	)

	_, err = j.Generate("test")
	if err != nil {
		t.Fatalf("Generate returned %v error, expected nil", err)
	}
}

func TestInspect(t *testing.T) {
	pubKey, err := ioutil.ReadFile("test/sample_key.pub")
	if err != nil {
		t.Fatalf("Unable to read public key: %v", err)
	}
	privKey, err := ioutil.ReadFile("test/sample_key")
	if err != nil {
		t.Fatalf("Unable to read private key: %v", err)
	}

	j := NewTokenProvider(
		token.WithPublicKey(string(pubKey)),
		token.WithPrivateKey(string(privKey)),
	)

	t.Run("Valid token", func(t *testing.T) {
		md := map[string]string{"foo": "bar"}
		roles := []string{"admin"}
		subject := "test"

		opts := []token.GenerateOption{
			token.WithMetadata(md),
			token.WithRoles(roles...),
		}

		tok, err := j.Generate(subject, opts...)
		if err != nil {
			t.Fatalf("Generate returned %v error, expected nil", err)
		}

		tok2, err := j.Inspect(tok.Token)
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
		tok, err := j.Generate("foo", token.WithExpiry(-10*time.Second))
		if err != nil {
			t.Fatalf("Generate returned %v error, expected nil", err)
		}

		if _, err = j.Inspect(tok.Token); err != token.ErrInvalidToken {
			t.Fatalf("Inspect returned %v error, expected %v", err, token.ErrInvalidToken)
		}
	})

	t.Run("Invalid token", func(t *testing.T) {
		_, err := j.Inspect("Invalid token")
		if err != token.ErrInvalidToken {
			t.Fatalf("Inspect returned %v error, expected %v", err, token.ErrInvalidToken)
		}
	})

}
