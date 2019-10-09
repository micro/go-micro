// Package acme abstracts away various ACME libraries
package acme

import (
	"errors"
	"net"
)

var (
	ErrProviderNotImplemented = errors.New("Provider not implemented")
)

// Provider is a ACME provider interface
type Provider interface {
	NewListener(...string) (net.Listener, error)
}
