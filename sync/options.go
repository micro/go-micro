package sync

import (
	"context"
	"crypto/tls"
	"time"
)

// Nodes sets the addresses to use
func Nodes(a ...string) Option {
	return func(o *Options) {
		o.Nodes = a
	}
}

// Prefix sets a prefix to any lock ids used
func Prefix(p string) Option {
	return func(o *Options) {
		o.Prefix = p
	}
}

// LockTTL sets the lock ttl
func LockTTL(t time.Duration) LockOption {
	return func(o *LockOptions) {
		o.TTL = t
	}
}

// LockWait sets the wait time
func LockWait(t time.Duration) LockOption {
	return func(o *LockOptions) {
		o.Wait = t
	}
}

// WithTLS sets the TLS config
func WithTLS(t *tls.Config) Option {
	return func(o *Options) {
		o.TLSConfig = t
	}
}

// WithContext sets the syncs context, for any extra configuration
func WithContext(c context.Context) Option {
	return func(o *Options) {
		o.Context = c
	}
}
