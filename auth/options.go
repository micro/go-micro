package auth

import (
	"time"

	"github.com/micro/go-micro/v2/auth/provider"
	"github.com/micro/go-micro/v2/store"
)

type Options struct {
	// Token is an auth token
	Token string
	// Public key base64 encoded
	PublicKey string
	// Private key base64 encoded
	PrivateKey string
	// Provider is an auth provider
	Provider provider.Provider
	// LoginURL is the relative url path where a user can login
	LoginURL string
	// Store to back auth
	Store store.Store
}

type Option func(o *Options)

// Store to back auth
func Store(s store.Store) Option {
	return func(o *Options) {
		o.Store = s
	}
}

// PublicKey is the JWT public key
func PublicKey(key string) Option {
	return func(o *Options) {
		o.PublicKey = key
	}
}

// PrivateKey is the JWT private key
func PrivateKey(key string) Option {
	return func(o *Options) {
		o.PrivateKey = key
	}
}

// ServiceToken sets an auth token
func ServiceToken(t string) Option {
	return func(o *Options) {
		o.Token = t
	}
}

// Provider set the auth provider
func Provider(p provider.Provider) Option {
	return func(o *Options) {
		o.Provider = p
	}
}

// LoginURL sets the auth LoginURL
func LoginURL(url string) Option {
	return func(o *Options) {
		o.LoginURL = url
	}
}

type GenerateOptions struct {
	// Metadata associated with the account
	Metadata map[string]string
	// Roles/scopes associated with the account
	Roles []string
	// SecretExpiry is the time the secret should live for
	SecretExpiry time.Duration
	// Namespace the account belongs too
	Namespace string
}

type GenerateOption func(o *GenerateOptions)

// WithMetadata for the generated account
func WithMetadata(md map[string]string) GenerateOption {
	return func(o *GenerateOptions) {
		o.Metadata = md
	}
}

// WithRoles for the generated account
func WithRoles(rs ...string) GenerateOption {
	return func(o *GenerateOptions) {
		o.Roles = rs
	}
}

// WithNamespace for the generated account
func WithNamespace(n string) GenerateOption {
	return func(o *GenerateOptions) {
		o.Namespace = n
	}
}

// WithSecretExpiry for the generated account's secret expires
func WithSecretExpiry(ex time.Duration) GenerateOption {
	return func(o *GenerateOptions) {
		o.SecretExpiry = ex
	}
}

// NewGenerateOptions from a slice of options
func NewGenerateOptions(opts ...GenerateOption) GenerateOptions {
	var options GenerateOptions
	for _, o := range opts {
		o(&options)
	}

	// set defualt expiry of secret
	if options.SecretExpiry == 0 {
		options.SecretExpiry = time.Hour * 24 * 7
	}

	return options
}

type TokenOptions struct {
	// TokenExpiry is the time the token should live for
	TokenExpiry time.Duration
}

type TokenOption func(o *TokenOptions)

// WithTokenExpiry for the token
func WithTokenExpiry(ex time.Duration) TokenOption {
	return func(o *TokenOptions) {
		o.TokenExpiry = ex
	}
}

// NewTokenOptions from a slice of options
func NewTokenOptions(opts ...TokenOption) TokenOptions {
	var options TokenOptions
	for _, o := range opts {
		o(&options)
	}

	// set defualt expiry of token
	if options.TokenExpiry == 0 {
		options.TokenExpiry = time.Minute
	}

	return options
}
