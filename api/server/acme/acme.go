// Package acme abstracts away various ACME libraries
package acme

import (
	"errors"
	"net"

	"golang.org/x/crypto/acme/autocert"
)

// ErrUnsupportedLibrary is returned by acme.New() if you
// attempt to instantiate a library we don't support
var ErrUnsupportedLibrary = errors.New("Unsupported Library")

const (
	// LibAutoCert is the acme library from golang.org/x/crypto/acme/autocert
	LibAutoCert = "autocert"
	// LibCertMagic is the acme library from github.com/mholt/certmagic
	LibCertMagic = "certmagic"
)

// Library is an provider interface
type Library interface {
	NewListener([]string) (net.Listener, error)
}

// AutoCert is the ACME provider from golang.org/x/crypto/acme/autocert
type AutoCert struct{}

// NewListener implements acme.Library
func (a *AutoCert) NewListener(ACMEHosts []string) (net.Listener, error) {
	return autocert.NewListener(ACMEHosts...), nil
}

// Default returns the autocert Library
func Default() Library {
	l, _ := New(LibAutoCert)
	return l
}

// New returns the library you've chosen, or an error. If an error is
// returned, it will be of the type acme.ErrUnsupportedLibrary
func New(lib string) (Library, error) {
	switch lib {
	case "autocert":
		return &AutoCert{}, nil
	default:
		return nil, ErrUnsupportedLibrary
	}
}
