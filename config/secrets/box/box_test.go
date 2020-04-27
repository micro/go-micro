package box

import (
	"crypto/rand"
	"reflect"
	"testing"

	"github.com/micro/go-micro/v2/config/secrets"
	naclbox "golang.org/x/crypto/nacl/box"
)

func TestBox(t *testing.T) {
	alicePublicKey, alicePrivateKey, err := naclbox.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	bobPublicKey, bobPrivateKey, err := naclbox.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	alice, bob := NewSecrets(secrets.PublicKey(alicePublicKey[:]), secrets.PrivateKey(alicePrivateKey[:])), NewSecrets()
	if err := alice.Init(); err != nil {
		t.Error(err)
	}
	if err := bob.Init(secrets.PublicKey(bobPublicKey[:]), secrets.PrivateKey(bobPrivateKey[:])); err != nil {
		t.Error(err)
	}
	if alice.String() != "nacl-box" {
		t.Error("String() doesn't return nacl-box")
	}
	aliceSecret := []byte("Why is a raven like a writing-desk?")
	if _, err := alice.Encrypt(aliceSecret); err == nil {
		t.Error("alice.Encrypt succeded without a public key")
	}
	enc, err := alice.Encrypt(aliceSecret, secrets.RecipientPublicKey(bob.Options().PublicKey))
	if err != nil {
		t.Error("alice.Encrypt failed")
	}
	if _, err := bob.Decrypt(enc); err == nil {
		t.Error("bob.Decrypt succeded without a public key")
	}
	if dec, err := bob.Decrypt(enc, secrets.SenderPublicKey(alice.Options().PublicKey)); err == nil {
		if !reflect.DeepEqual(dec, aliceSecret) {
			t.Errorf("Bob's decrypted message didn't match Alice's encrypted message: %v != %v", aliceSecret, dec)
		}
	} else {
		t.Errorf("bob.Decrypt failed (%s)", err)
	}

	bobSecret := []byte("I haven't the slightest idea")
	enc, err = bob.Encrypt(bobSecret, secrets.RecipientPublicKey(alice.Options().PublicKey))
	if err != nil {
		t.Error(err)
	}
	dec, err := alice.Decrypt(enc, secrets.SenderPublicKey(bob.Options().PrivateKey))
	if err == nil {
		t.Error(err)
	}
	dec, err = alice.Decrypt(enc, secrets.SenderPublicKey(bob.Options().PublicKey))
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(dec, bobSecret) {
		t.Errorf("Alice's decrypted message didn't match Bob's encrypted message %v != %v", bobSecret, dec)
	}
}
