package tunnel

import (
	"bytes"
	"testing"
)

func TestEncrypt(t *testing.T) {
	key := []byte("tokenpassphrase")
	gcm, err := newCipher(key)
	if err != nil {
		t.Fatal(err)
	}

	data := []byte("supersecret")

	cipherText, err := Encrypt(gcm, data)
	if err != nil {
		t.Errorf("failed to encrypt data: %v", err)
	}

	// verify the cipherText is not the same as data
	if bytes.Equal(data, cipherText) {
		t.Error("encrypted data are the same as plaintext")
	}
}

func TestDecrypt(t *testing.T) {
	key := []byte("tokenpassphrase")
	gcm, err := newCipher(key)
	if err != nil {
		t.Fatal(err)
	}

	data := []byte("supersecret")

	cipherText, err := Encrypt(gcm, data)
	if err != nil {
		t.Errorf("failed to encrypt data: %v", err)
	}

	plainText, err := Decrypt(gcm, cipherText)
	if err != nil {
		t.Errorf("failed to decrypt data: %v", err)
	}

	// verify the plainText is the same as data
	if !bytes.Equal(data, plainText) {
		t.Error("decrypted data not the same as plaintext")
	}
}
