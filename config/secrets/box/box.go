// Package box is an asymmetric implementation of config/secrets using nacl/box
package box

import "github.com/micro/go-micro/v2/config/secrets"

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
func (b *box) Init(...secrets.Option) error {
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

// Encrypt encrypts
func (b *box) Encrypt(in []byte, opts ...secrets.EncryptOption) ([]byte, error) {
	return nil, nil
}

// Decrypt Decrypts
func (b *box) Decrypt(in []byte, opts ...secrets.DecryptOption) ([]byte, error) {
	return nil, nil
}
