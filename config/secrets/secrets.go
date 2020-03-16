// Package secrets is an interface for encrypting and decrypting secrets
package secrets

import "context"

// Codec encrypts or decrypts arbitrary data. The data should be as small as possible
type Codec interface {
	Init(...Option) error
	Options() Options
	String() string
	Decrypt([]byte, ...DecryptOption) ([]byte, error)
	Encrypt([]byte, ...EncryptOption) ([]byte, error)
}

// Options is a codec's options
// SecretKey or both PublicKey and PrivateKey should be set depending on the
// underlying implementation
type Options struct {
	SecretKey  []byte
	PrivateKey []byte
	PublicKey  []byte
	Context    context.Context
}

// Option sets options
type Option func(*Options)

// SecretKey sets the symmetric secret key
func SecretKey(key []byte) Option {
	return func(o *Options) {
		o.SecretKey = make([]byte, len(key))
		copy(o.SecretKey, key)
	}
}

// PublicKey sets the asymmetric Public Key of this codec
func PublicKey(key []byte) Option {
	return func(o *Options) {
		o.PublicKey = make([]byte, len(key))
		copy(o.PublicKey, key)
	}
}

// PrivateKey sets the asymmetric Private Key of this codec
func PrivateKey(key []byte) Option {
	return func(o *Options) {
		o.PrivateKey = make([]byte, len(key))
		copy(o.PrivateKey, key)
	}
}

// DecryptOptions can be passed to Codec.Decrypt
type DecryptOptions struct {
	SenderPublicKey []byte
}

// DecryptOption sets DecryptOptions
type DecryptOption func(*DecryptOptions)

// SenderPublicKey is the Public Key of the Codec that encrypted this message
func SenderPublicKey(key []byte) DecryptOption {
	return func(d *DecryptOptions) {
		d.SenderPublicKey = make([]byte, len(key))
		copy(d.SenderPublicKey, key)
	}
}

// EncryptOptions can be passed to Codec.Encrypt
type EncryptOptions struct {
	RecipientPublicKey []byte
}

// EncryptOption Sets EncryptOptions
type EncryptOption func(*EncryptOptions)

// RecipientPublicKey is the Public Key of the Codec that will decrypt this message
func RecipientPublicKey(key []byte) EncryptOption {
	return func(e *EncryptOptions) {
		e.RecipientPublicKey = make([]byte, len(key))
		copy(e.RecipientPublicKey, key)
	}
}
