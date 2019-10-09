// Package acme abstracts away various ACME libraries
package acme

import (
	"errors"
	"net"
)

const (
	// LibAutoCert is the acme library from golang.org/x/crypto/acme/autocert
	LibAutoCert = "autocert"
	// LibCertMagic is the acme library from github.com/mholt/certmagic
	LibCertMagic = "certmagic"
)

var (
	ErrLibraryNotImplemented = errors.New("Library not implemented")
)

// Library is a ACME provider interface
type Library interface {
	NewListener(...string) (net.Listener, error)
}
