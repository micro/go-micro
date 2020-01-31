package auth

import (
	b64 "encoding/base64"
)

type Options struct {
	PublicKey  []byte
	PrivateKey []byte
}

type Option func(o *Options)

// PublicKey is the JWT public key
func PublicKey(key string) Option {
	return func(o *Options) {
		o.PublicKey, _ = b64.StdEncoding.DecodeString(key)
	}
}

// PrivateKey is the JWT private key
func PrivateKey(key string) Option {
	return func(o *Options) {
		o.PrivateKey, _ = b64.StdEncoding.DecodeString(key)
	}
}
