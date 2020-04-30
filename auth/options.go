package auth

import (
	"time"

	"github.com/micro/go-micro/v2/auth/provider"
	"github.com/micro/go-micro/v2/store"
)

func NewOptions(opts ...Option) Options {
	var options Options
	for _, o := range opts {
		o(&options)
	}

	if len(options.Namespace) == 0 {
		options.Namespace = DefaultNamespace
	}

	return options
}

type Options struct {
	// Namespace the service belongs to
	Namespace string
	// ID is the services auth ID
	ID string
	// Secret is used to authenticate the service
	Secret string
	// Token is the services token used to authenticate itself
	Token *Token
	// PublicKey for decoding JWTs
	PublicKey string
	// PrivateKey for encoding JWTs
	PrivateKey string
	// Provider is an auth provider
	Provider provider.Provider
	// LoginURL is the relative url path where a user can login
	LoginURL string
	// Store to back auth
	Store store.Store
}

type Option func(o *Options)

// Namespace the service belongs to
func Namespace(n string) Option {
	return func(o *Options) {
		o.Namespace = n
	}
}

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

// Credentials sets the auth credentials
func Credentials(id, secret string) Option {
	return func(o *Options) {
		o.ID = id
		o.Secret = secret
	}
}

// ClientToken sets the auth token to use when making requests
func ClientToken(token *Token) Option {
	return func(o *Options) {
		o.Token = token
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
	// Namespace the account belongs too
	Namespace string
	// Provider of the account, e.g. oauth
	Provider string
	// Type of the account, e.g. user
	Type string
	// Secret used to authenticate the account
	Secret string
}

type GenerateOption func(o *GenerateOptions)

// WithSecret for the generated account
func WithSecret(s string) GenerateOption {
	return func(o *GenerateOptions) {
		o.Secret = s
	}
}

// WithType for the generated account
func WithType(t string) GenerateOption {
	return func(o *GenerateOptions) {
		o.Type = t
	}
}

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

// WithProvider for the generated account
func WithProvider(p string) GenerateOption {
	return func(o *GenerateOptions) {
		o.Provider = p
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

type TokenOptions struct {
	// ID for the account
	ID string
	// Secret for the account
	Secret string
	// RefreshToken is used to refesh a token
	RefreshToken string
	// Expiry is the time the token should live for
	Expiry time.Duration
}

type TokenOption func(o *TokenOptions)

// WithExpiry for the token
func WithExpiry(ex time.Duration) TokenOption {
	return func(o *TokenOptions) {
		o.Expiry = ex
	}
}

func WithCredentials(id, secret string) TokenOption {
	return func(o *TokenOptions) {
		o.ID = id
		o.Secret = secret
	}
}

func WithToken(rt string) TokenOption {
	return func(o *TokenOptions) {
		o.RefreshToken = rt
	}
}

// NewTokenOptions from a slice of options
func NewTokenOptions(opts ...TokenOption) TokenOptions {
	var options TokenOptions
	for _, o := range opts {
		o(&options)
	}

	// set defualt expiry of token
	if options.Expiry == 0 {
		options.Expiry = time.Minute
	}

	return options
}
