package registry

import (
	"crypto/tls"
	"time"

	"golang.org/x/net/context"
)

type Options struct {
	Addrs                   []string
	Timeout                 time.Duration
	Secure                  bool
	TLSConfig               *tls.Config
	OAuth2ClientCredentials *oauth2ClientCredentials

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type RegisterOptions struct {
	TTL time.Duration
	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

// Options for OAuth 2.0 Client Credentials Grant Flow
type oauth2ClientCredentials struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
}

// Addrs is the registry addresses to use
func Addrs(addrs ...string) Option {
	return func(o *Options) {
		o.Addrs = addrs
	}
}

func Timeout(t time.Duration) Option {
	return func(o *Options) {
		o.Timeout = t
	}
}

// Secure communication with the registry
func Secure(b bool) Option {
	return func(o *Options) {
		o.Secure = b
	}
}

// Specify TLS Config
func TLSConfig(t *tls.Config) Option {
	return func(o *Options) {
		o.TLSConfig = t
	}
}

// Enable OAuth 2.0 Client Credentials Grant Flow
func OAuth2ClientCredentials(clientID, clientSecret, tokenURL string) Option {
	return func(o *Options) {
		o.OAuth2ClientCredentials = &oauth2ClientCredentials{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			TokenURL:     tokenURL,
		}
	}
}

func RegisterTTL(t time.Duration) RegisterOption {
	return func(o *RegisterOptions) {
		o.TTL = t
	}
}
