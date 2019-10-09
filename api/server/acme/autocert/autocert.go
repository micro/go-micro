// Package autocert is the ACME interpreter from golang.org/x/crypto/acme/autocert
package autocert

import (
	"net"

	"github.com/micro/go-micro/api/server/acme"
	"golang.org/x/crypto/acme/autocert"
)

// autoCertACME is the ACME provider from golang.org/x/crypto/acme/autocert
type autocertACME struct{}

// NewListener implement acme.Library
func (a *autocertACME) NewListener(ACMEHosts ...string) (net.Listener, error) {
	return autocert.NewListener(ACMEHosts...), nil
}

// New returns an autocert acme.Library
func New() acme.Library {
	return &autocertACME{}
}
