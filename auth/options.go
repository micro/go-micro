package auth

import (
	"time"

	"github.com/micro/go-micro/v2/auth/provider"
)

type Options struct {
	// Token is an auth token
	Token string
	// Public key base64 encoded
	PublicKey string
	// Private key base64 encoded
	PrivateKey string
	// Endpoints to exclude
	Exclude []string
	// Provider is an auth provider
	Provider provider.Provider
	// LoginURL is the relative url path where a user can login
	LoginURL string
}

type Option func(o *Options)

// Exclude ecludes a set of endpoints from authorization
func Exclude(e ...string) Option {
	return func(o *Options) {
		o.Exclude = e
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

// Token sets an auth token
func Token(t string) Option {
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
	Roles []*Role
	//Expiry of the token
	Expiry time.Time
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

// Expiry for the generated account's token expires
func Expiry(ex time.Time) func(o *GenerateOptions) {
	return func(o *GenerateOptions) {
		o.Expiry = ex
	}
}

// NewGenerateOptions from a slice of options
func NewGenerateOptions(opts ...GenerateOption) GenerateOptions {
	var options GenerateOptions
	for _, o := range opts {
		o(&options)
	}
	//set defualt expiry of token
	if options.Expiry.IsZero() {
		options.Expiry = time.Now().Add(time.Hour * 24)
	}
	return options
}
