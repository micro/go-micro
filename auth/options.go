package auth

import (
	b64 "encoding/base64"
)

type Options struct {
	PublicKey  []byte
	PrivateKey []byte
	Excludes   []string
}

type Option func(o *Options)

// Excludes endpoints from auth
func Excludes(excludes ...string) Option {
	return func(o *Options) {
		o.Excludes = excludes
	}
}

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

type GenerateOptions struct {
	Metadata map[string]string
	Roles    []*Role
}

type GenerateOption func(o *GenerateOptions)

// Metadata for the generated account
func Metadata(md map[string]string) func(o *GenerateOptions) {
	return func(o *GenerateOptions) {
		o.Metadata = md
	}
}

// Roles for the generated account
func Roles(rs []*Role) func(o *GenerateOptions) {
	return func(o *GenerateOptions) {
		o.Roles = rs
	}
}

// NewGenerateOptions from a slice of options
func NewGenerateOptions(opts ...GenerateOption) GenerateOptions {
	var options GenerateOptions
	for _, o := range opts {
		o(&options)
	}

	return options
}
