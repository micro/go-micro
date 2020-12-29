// Package certmagic is the ACME provider from github.com/caddyserver/certmagic
package certmagic

import (
	"crypto/tls"
	"math/rand"
	"net"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/micro/go-micro/v2/api/server/acme"
	"github.com/micro/go-micro/v2/logger"
)

type certmagicProvider struct {
	opts acme.Options
}

// TODO: set self-contained options
func (c *certmagicProvider) setup() {
	certmagic.DefaultACME.CA = c.opts.CA
	if c.opts.ChallengeProvider != nil {
		// Enabling DNS Challenge disables the other challenges
		certmagic.DefaultACME.DNSProvider = c.opts.ChallengeProvider
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
	// RenewalWindowRatio [0.33 - 0.50)
	rand.Seed(time.Now().UnixNano())
	randomRatio := float64(rand.Intn(17)+33) * 0.01
	certmagic.Default.RenewalWindowRatio = randomRatio
}

func (c *certmagicProvider) Listen(hosts ...string) (net.Listener, error) {
	c.setup()
	return certmagic.Listen(hosts)
}

func (c *certmagicProvider) TLSConfig(hosts ...string) (*tls.Config, error) {
	c.setup()
	return certmagic.TLS(hosts)
}

// NewProvider returns a certmagic provider
func NewProvider(options ...acme.Option) acme.Provider {
	opts := acme.DefaultOptions()

	for _, o := range options {
		o(&opts)
	}

	if opts.Cache != nil {
		if _, ok := opts.Cache.(certmagic.Storage); !ok {
			logger.Fatal("ACME: cache provided doesn't implement certmagic's Storage interface")
		}
	}

	return &certmagicProvider{
		opts: opts,
	}
}
