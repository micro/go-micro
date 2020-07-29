package auth

import (
	"context"
	"time"

	"github.com/micro/go-micro/v3/store"
)

func NewOptions(opts ...Option) Options {
	var options Options
	for _, o := range opts {
		o(&options)
	}
	return options
}

type Options struct {
	// Issuer of the service's account
	Issuer string
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
	// LoginURL is the relative url path where a user can login
	LoginURL string
	// Store to back auth
	Store store.Store
	// Addrs sets the addresses of auth
	Addrs []string
	// Context to store other options
	Context context.Context
}

type Option func(o *Options)

// Addrs is the auth addresses to use
func Addrs(addrs ...string) Option {
	return func(o *Options) {
		o.Addrs = addrs
	}
}

// Issuer of the services account
func Issuer(i string) Option {
	return func(o *Options) {
		o.Issuer = i
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

// LoginURL sets the auth LoginURL
func LoginURL(url string) Option {
	return func(o *Options) {
		o.LoginURL = url
	}
}

type GenerateOptions struct {
	// Metadata associated with the account
	Metadata map[string]string
	// Scopes the account has access too
	Scopes []string
	// Provider of the account, e.g. oauth
	Provider string
	// Type of the account, e.g. user
	Type string
	// Secret used to authenticate the account
	Secret string
	// Issuer of the account, e.g. micro
	Issuer string
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

// WithProvider for the generated account
func WithProvider(p string) GenerateOption {
	return func(o *GenerateOptions) {
		o.Provider = p
	}
}

// WithScopes for the generated account
func WithScopes(s ...string) GenerateOption {
	return func(o *GenerateOptions) {
		o.Scopes = s
	}
}

// WithIssuer for the generated account
func WithIssuer(i string) GenerateOption {
	return func(o *GenerateOptions) {
		o.Issuer = i
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
	// Issuer of the account
	Issuer string
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

func WithTokenIssuer(iss string) TokenOption {
	return func(o *TokenOptions) {
		o.Issuer = iss
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

type VerifyOptions struct {
	Context   context.Context
	Namespace string
}

type VerifyOption func(o *VerifyOptions)

func VerifyContext(ctx context.Context) VerifyOption {
	return func(o *VerifyOptions) {
		o.Context = ctx
	}
}
func VerifyNamespace(ns string) VerifyOption {
	return func(o *VerifyOptions) {
		o.Namespace = ns
	}
}

type RulesOptions struct {
	Context   context.Context
	Namespace string
}

type RulesOption func(o *RulesOptions)

func RulesContext(ctx context.Context) RulesOption {
	return func(o *RulesOptions) {
		o.Context = ctx
	}
}

func RulesNamespace(ns string) RulesOption {
	return func(o *RulesOptions) {
		o.Namespace = ns
	}
}
