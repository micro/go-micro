package registry

import (
	"context"

	"github.com/miekg/dns"
)

type mdnsDomainKey struct{}

// helper for setting registry options
func setRegistryOption(k, v interface{}) Option {
	return func(o *Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

// Specify domain for mdns registry
func Domain(d string) Option {
	return func(o *Options) {
		setRegistryOption(mdnsDomainKey{}, dns.Fqdn(d))
	}
}
