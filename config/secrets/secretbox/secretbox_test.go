package secretbox

import (
	"encoding/base64"
	"reflect"
	"testing"

	"github.com/micro/go-micro/v2/config/secrets"
)

func TestSecretBox(t *testing.T) {
	secretKey, err := base64.StdEncoding.DecodeString("4jbVgq8FsAV7vy+n8WqEZrl7BUtNqh3fYT5RXzXOPFY=")
	if err != nil {
		t.Fatal(err)
	}

	s := NewSecrets()

	if err := s.Init(); err == nil {
		t.Error("Secretbox accepted an empty secret key")
	}
	if err := s.Init(secrets.Key([]byte("invalid"))); err == nil {
		t.Error("Secretbox accepted a secret key that is invalid")
	}

	if err := s.Init(secrets.Key(secretKey)); err != nil {
		t.Fatal(err)
	}

	o := s.Options()
	if !reflect.DeepEqual(o.Key, secretKey) {
		t.Error("Init() didn't set secret key correctly")
	}
	if s.String() != "nacl-secretbox" {
		t.Error(s.String() + " should be nacl-secretbox")
	}

	// Try 10 times to get different nonces
	for i := 0; i < 10; i++ {
		message := []byte(`Can you hear me, Major Tom?`)

		encrypted, err := s.Encrypt(message)
		if err != nil {
			t.Errorf("Failed to encrypt message (%s)", err)
		}

		decrypted, err := s.Decrypt(encrypted)
		if err != nil {
			t.Errorf("Failed to decrypt encrypted message (%s)", err)
		}

		if !reflect.DeepEqual(message, decrypted) {
			t.Errorf("Decrypted Message dod not match encrypted message")
		}
	}
}
