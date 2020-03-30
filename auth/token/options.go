package token

import (
	"time"

	"github.com/micro/go-micro/v2/store"
)

type Options struct {
	// Store to persist the tokens
	Store store.Store
	// PublicKey base64 encoded, used by JWT
	PublicKey string
	// PrivateKey base64 encoded, used by JWT
	PrivateKey string
}

type Option func(o *Options)

// WithStore sets the token providers store
func WithStore(s store.Store) Option {
	return func(o *Options) {
		o.Store = s
	}
}

// WithPublicKey sets the JWT public key
func WithPublicKey(key string) Option {
	return func(o *Options) {
		o.PublicKey = key
	}
}

// WithPrivateKey sets the JWT private key
func WithPrivateKey(key string) Option {
	return func(o *Options) {
		o.PrivateKey = key
	}
}

func NewOptions(opts ...Option) Options {
	var options Options
	for _, o := range opts {
		o(&options)
	}
	//set default store
	if options.Store == nil {
		options.Store = store.DefaultStore
	}
	return options
}

type GenerateOptions struct {
	// Expiry for the token
	Expiry time.Duration
	// Metadata associated with the account
	Metadata map[string]string
	// Roles/scopes associated with the account
	Roles []string
	// Namespace the account belongs too
	Namespace string
}

type GenerateOption func(o *GenerateOptions)

// WithExpiry for the generated account's token expires
func WithExpiry(d time.Duration) GenerateOption {
	return func(o *GenerateOptions) {
		o.Expiry = d
	}
}

// WithMetadata for the token
func WithMetadata(md map[string]string) func(o *GenerateOptions) {
	return func(o *GenerateOptions) {
		o.Metadata = md
	}
}

// WithRoles for the token
func WithRoles(rs ...string) func(o *GenerateOptions) {
	return func(o *GenerateOptions) {
		o.Roles = rs
	}
}

// WithNamespace for the token
func WithNamespace(n string) func(o *GenerateOptions) {
	return func(o *GenerateOptions) {
		o.Namespace = n
	}
}

// NewGenerateOptions from a slice of options
func NewGenerateOptions(opts ...GenerateOption) GenerateOptions {
	var options GenerateOptions
	for _, o := range opts {
		o(&options)
	}
	//set default Expiry of token
	if options.Expiry == 0 {
		options.Expiry = time.Minute * 15
	}
	return options
}
