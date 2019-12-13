// Package acme abstracts away various ACME libraries
package acme

import (
	"errors"
	"net"
)

var (
	// ErrProviderNotImplemented can be returned when attempting to
	// instantiate an unimplemented provider
	ErrProviderNotImplemented = errors.New("Provider not implemented")
)

// Provider is a ACME provider interface
type Provider interface {
	NewListener(...string) (net.Listener, error)
}

// The Let's Encrypt ACME endpoints
const (
	LetsEncryptStagingCA    = "https://acme-staging-v02.api.letsencrypt.org/directory"
	LetsEncryptProductionCA = "https://acme-v02.api.letsencrypt.org/directory"
)
