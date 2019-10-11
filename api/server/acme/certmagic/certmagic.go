// Package certmagic is the ACME provider from github.com/mholt/certmagic
package certmagic

import (
	"net"

	"github.com/mholt/certmagic"

	"github.com/micro/go-micro/api/server/acme"
)

type certmagicProvider struct {
	opts *acme.Options
}

func (c *certmagicProvider) NewListener(ACMEHosts ...string) (net.Listener, error) {
	if c.opts.ChallengeProvider != nil {
		// Enabling DNS Challenge disables the other challenges
		certmagic.Default.DNSProvider = c.opts.ChallengeProvider
	}
	if c.opts.OnDemand {
		certmagic.Default.OnDemand = new(certmagic.OnDemandConfig)
	}
	return certmagic.Listen(ACMEHosts)
}

// New returns a certmagic provider
func New(options ...acme.Option) acme.Provider {
	o := &acme.Options{}
	if len(options) == 0 {
		for _, op := range acme.Default() {
			op(o)
		}
	} else {
		for _, op := range options {
			op(o)
		}
	}
	return &certmagicProvider{
		opts: o,
	}
}
