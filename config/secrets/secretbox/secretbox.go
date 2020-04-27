// Package secretbox is a config/secrets implementation that uses nacl/secretbox
// to do symmetric encryption / verification
package secretbox

import (
	"github.com/micro/go-micro/v2/config/secrets"
	"github.com/pkg/errors"
	"golang.org/x/crypto/nacl/secretbox"

	"crypto/rand"
)

const keyLength = 32

type secretBox struct {
	options secrets.Options

	secretKey [keyLength]byte
}

// NewSecrets returns a secretbox codec
func NewSecrets(opts ...secrets.Option) secrets.Secrets {
	sb := &secretBox{}
	for _, o := range opts {
		o(&sb.options)
	}
	return sb
}

func (s *secretBox) Init(opts ...secrets.Option) error {
	for _, o := range opts {
		o(&s.options)
	}
	if len(s.options.Key) == 0 {
		return errors.New("no secret key is defined")
	}
	if len(s.options.Key) != keyLength {
		return errors.Errorf("secret key must be %d bytes long", keyLength)
	}
	copy(s.secretKey[:], s.options.Key)
	return nil
}

func (s *secretBox) Options() secrets.Options {
	return s.options
}

func (s *secretBox) String() string {
	return "nacl-secretbox"
}

func (s *secretBox) Encrypt(in []byte, opts ...secrets.EncryptOption) ([]byte, error) {
	// no opts are expected, so they are ignored

	// there must be a unique nonce for each message
	var nonce [24]byte
	if _, err := rand.Reader.Read(nonce[:]); err != nil {
		return []byte{}, errors.Wrap(err, "couldn't obtain a random nonce from crypto/rand")
	}
	return secretbox.Seal(nonce[:], in, &nonce, &s.secretKey), nil
}

func (s *secretBox) Decrypt(in []byte, opts ...secrets.DecryptOption) ([]byte, error) {
	// no options are expected, so they are ignored

	var decryptNonce [24]byte
	copy(decryptNonce[:], in[:24])
	decrypted, ok := secretbox.Open(nil, in[24:], &decryptNonce, &s.secretKey)
	if !ok {
		return []byte{}, errors.New("decryption failed (is the key set correctly?)")
	}
	return decrypted, nil
}
