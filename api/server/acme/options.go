package acme

import "github.com/go-acme/lego/v3/challenge"

// Option (or Options) are passed to New() to configure providers
type Option func(o *Options)

// Options represents various options you can present to ACME providers
type Options struct {
	// AcceptTLS must be set to true to indicate that you have read your
	// provider's terms of service.
	AcceptToS bool
	// CA is the CA to use
	CA string
	// ChallengeProvider is a go-acme/lego challenge provider. Set this if you
	// want to use DNS Challenges. Otherwise, tls-alpn-01 will be used
	ChallengeProvider challenge.Provider
	// Issue certificates for domains on demand. Otherwise, certs will be
	// retrieved / issued on start-up.
	OnDemand bool
	// Cache is a storage interface. Most ACME libraries have an cache, but
	// there's no defined interface, so if you consume this option
	// sanity check it before using.
	Cache interface{}
}

// AcceptToS indicates whether you accept your CA's terms of service
func AcceptToS(b bool) Option {
	return func(o *Options) {
		o.AcceptToS = b
	}
}

// CA sets the CA of an acme.Options
func CA(CA string) Option {
	return func(o *Options) {
		o.CA = CA
	}
}

// ChallengeProvider sets the Challenge provider of an acme.Options
// if set, it enables the DNS challenge, otherwise tls-alpn-01 will be used.
func ChallengeProvider(p challenge.Provider) Option {
	return func(o *Options) {
		o.ChallengeProvider = p
	}
}

// OnDemand enables on-demand certificate issuance. Not recommended for use
// with the DNS challenge, as the first connection may be very slow.
func OnDemand(b bool) Option {
	return func(o *Options) {
		o.OnDemand = b
	}
}

// Cache provides a cache / storage interface to the underlying ACME library
// as there is no standard, this needs to be validated by the underlying
// implentation.
func Cache(c interface{}) Option {
	return func(o *Options) {
		o.Cache = c
	}
}

// DefaultOptions uses the Let's Encrypt Production CA, with DNS Challenge disabled.
func DefaultOptions() Options {
	return Options{
		AcceptToS: true,
		CA:        LetsEncryptProductionCA,
		OnDemand:  true,
	}
}
