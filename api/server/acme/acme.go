// Package acme abstracts away various ACME libraries
package acme

import (
	"errors"
	"net"
)

const (
	// Autocert is the acme provider from golang.org/x/crypto/acme/autocert
	Autocert = "autocert"
	// Certmagic is the acme provider from github.com/mholt/certmagic
	Certmagic = "certmagic"
)

var (
	ErrProviderNotImplemented = errors.New("Provider not implemented")
)

// Provider is a ACME provider interface
type Provider interface {
	NewListener(...string) (net.Listener, error)
}
