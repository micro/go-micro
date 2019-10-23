// Package certmagic is the ACME provider from github.com/mholt/certmagic
package certmagic

import (
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/mholt/certmagic"

	"github.com/micro/go-micro/api/server/acme"
)

type certmagicProvider struct {
	opts *acme.Options
}

func (c *certmagicProvider) NewListener(ACMEHosts ...string) (net.Listener, error) {
	certmagic.Default.CA = c.opts.CA
	if c.opts.ChallengeProvider != nil {
		// Enabling DNS Challenge disables the other challenges
		certmagic.Default.DNSProvider = c.opts.ChallengeProvider
	}
	if c.opts.OnDemand {
		certmagic.Default.OnDemand = new(certmagic.OnDemandConfig)
	}
	if c.opts.Cache != nil {
		// already validated by new()
		certmagic.Default.Storage = c.opts.Cache.(certmagic.Storage)
	}
	// If multiple instances of the provider are running, inject some
	// randomness so they don't collide
	rand.Seed(time.Now().UnixNano())
	randomDuration := (7 * 24 * time.Hour) + (time.Duration(rand.Intn(504)) * time.Hour)
	certmagic.Default.RenewDurationBefore = randomDuration

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
	if o.Cache != nil {
		if _, ok := o.Cache.(certmagic.Storage); !ok {
			log.Fatal("ACME: cache provided doesn't implement certmagic's Storage interface")
		}
	}

	return &certmagicProvider{
		opts: o,
	}
}
