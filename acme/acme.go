// Package acme abstracts away various ACME libraries
package acme

import (
	"crypto/tls"
	"errors"
	"net"
)

// The Let's Encrypt ACME endpoints
const (
	LetsEncryptStagingCA    = "https://acme-staging-v02.api.letsencrypt.org/directory"
	LetsEncryptProductionCA = "https://acme-v02.api.letsencrypt.org/directory"
)

var (
	// ErrProviderNotImplemented can be returned when attempting to
	// instantiate an unimplemented provider
	ErrProviderNotImplemented = errors.New("Provider not implemented")
)

// Provider is a ACME provider interface
type Provider interface {
	// Listen returns a new listener
	Listen(...string) (net.Listener, error)
	// TLSConfig returns a tls config
	TLSConfig(...string) (*tls.Config, error)
	// Implementation of the acme provider
	String() string
}

// Challenge is used to create an acme dns challenge
type Challenge interface {
	Present(domain, token, key string) error
	Remove(domain, token, key string) error
}
