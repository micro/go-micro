package jwt

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/asim/go-micro/v3/auth"
	"github.com/asim/go-micro/v3/util/token"
)

func TestGenerate(t *testing.T) {
	privKey, err := ioutil.ReadFile("test/sample_key")
	if err != nil {
		t.Fatalf("Unable to read private key: %v", err)
	}

	j := NewTokenProvider(
		token.WithPrivateKey(string(privKey)),
	)

	_, err = j.Generate(&auth.Account{ID: "test"})
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
		scopes := []string{"admin"}
		subject := "test"
		name := "testname"

		acc := &auth.Account{ID: subject, Scopes: scopes, Metadata: md, Name: name}
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
		if tok2.Name != name {
			t.Errorf("Inspect returned %v as the token name, expected %v", tok2.Name, name)
		}
	})

	t.Run("Expired token", func(t *testing.T) {
		tok, err := j.Generate(&auth.Account{}, token.WithExpiry(-10*time.Second))
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

	t.Run("Default name", func(t *testing.T) {
		tok, err := j.Generate(&auth.Account{ID: "test"})
		if err != nil {
			t.Fatalf("Generate returned %v error, expected nil", err)
		}

		tok2, err := j.Inspect(tok.Token)
		if err != nil {
			t.Fatalf("Inspect returned %v error, expected nil", err)
		}
		if tok2.Name != "test" {
			t.Fatalf("Inspect returned %v as the token name, expected test", tok2.Name)
		}
	})

}
