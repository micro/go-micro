// Package box is an asymmetric implementation of config/secrets using nacl/box
package box

import (
	"github.com/micro/go-micro/v2/config/secrets"
	"github.com/pkg/errors"
	naclbox "golang.org/x/crypto/nacl/box"

	"crypto/rand"
)

type box struct {
	options secrets.Options

	publicKey  [32]byte
	privateKey [32]byte
}

// NewCodec returns a nacl-box codec
func NewCodec(opts ...secrets.Option) secrets.Codec {
	b := &box{}
	for _, o := range opts {
		o(&b.options)
	}
	return b
}

// Init initialises a box
func (b *box) Init(opts ...secrets.Option) error {
	for _, o := range opts {
		o(&b.options)
	}
	if len(b.options.PrivateKey) != 32 || len(b.options.PublicKey) != 32 {
		return errors.New("A 32 byte public and private key must be provided")
	}
	copy(b.privateKey[:], b.options.PrivateKey)
	copy(b.publicKey[:], b.options.PublicKey)
	return nil
}

// Options returns options
func (b *box) Options() secrets.Options {
	return b.options
}

// String returns nacl-box
func (*box) String() string {
	return "nacl-box"
}

// Encrypt encrypts a message with the sender's private key and the receipient's public key
func (b *box) Encrypt(in []byte, opts ...secrets.EncryptOption) ([]byte, error) {
	var options secrets.EncryptOptions
	for _, o := range opts {
		o(&options)
	}
	if len(options.RecipientPublicKey) != 32 {
		return []byte{}, errors.New("Recepient's public key must be provided")
	}
	var recipientPublicKey [32]byte
	copy(recipientPublicKey[:], options.RecipientPublicKey)
	var nonce [24]byte
	if _, err := rand.Reader.Read(nonce[:]); err != nil {
		return []byte{}, errors.Wrap(err, "Couldn't obtain a random nonce from crypto/rand")
	}
	return naclbox.Seal(nonce[:], in, &nonce, &recipientPublicKey, &b.privateKey), nil
}

// Decrypt Decrypts a message with the receiver's private key and the sender's public key
func (b *box) Decrypt(in []byte, opts ...secrets.DecryptOption) ([]byte, error) {
	var options secrets.DecryptOptions
	for _, o := range opts {
		o(&options)
	}
	if len(options.SenderPublicKey) != 32 {
		return []byte{}, errors.New("Sender's public key bust be provided")
	}
	var nonce [24]byte
	var senderPublicKey [32]byte
	copy(nonce[:], in[:24])
	copy(senderPublicKey[:], options.SenderPublicKey)
	decrypted, ok := naclbox.Open(nil, in[24:], &nonce, &senderPublicKey, &b.privateKey)
	if !ok {
		return []byte{}, errors.New("Incoming message couldn't be verified / decrypted")
	}
	return decrypted, nil
}
