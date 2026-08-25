// Package profile provides grouped plugin profiles for go-micro
package profile

import (
	"fmt"
	"sync"

	"go-micro.dev/v6/broker"
	"go-micro.dev/v6/events"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
	"go-micro.dev/v6/transport"
)

type Profile struct {
	Registry  registry.Registry
	Broker    broker.Broker
	Store     store.Store
	Transport transport.Transport
	Stream    events.Stream
}

// profiles holds the registered named profiles. Only zero-dependency profiles
// live in this package; profiles that pull plugin machinery (e.g. NATS)
// register themselves from their own subpackage so that importing this
// package — and therefore cmd/service — does not link them.
var (
	profilesMu sync.RWMutex
	profiles   = map[string]func() (Profile, error){
		"local": LocalProfile,
	}
)

// Register makes a named profile loadable. Plugin-backed profiles call this
// from their package init (see service/profile/natsprofile); importing that
// package — directly or via go-micro.dev/v6/cmd/defaults — is what makes the
// profile available. It panics on an empty name or nil constructor, which is
// the useful failure mode for init-time registration.
func Register(name string, fn func() (Profile, error)) {
	if name == "" {
		panic("profile: Register with empty name")
	}
	if fn == nil {
		panic("profile: Register " + name + " with nil constructor")
	}
	profilesMu.Lock()
	defer profilesMu.Unlock()
	profiles[name] = fn
}

// Load returns the named profile. When the profile is not registered the
// error covers both causes: a mistyped name, or a plugin profile whose
// registering package is not linked into the binary.
func Load(name string) (Profile, error) {
	profilesMu.RLock()
	fn, ok := profiles[name]
	profilesMu.RUnlock()
	if !ok {
		return Profile{}, fmt.Errorf(
			"profile %q is not registered: check the name, and for plugin profiles import go-micro.dev/v6/cmd/defaults (all plugins) or the profile's own package to link it",
			name,
		)
	}
	return fn()
}

// LocalProfile returns a profile with local mDNS as the registry, HTTP as the broker, file as the store, and HTTP as the transport
// It is used for local development and testing
func LocalProfile() (Profile, error) {
	stream, err := events.NewStream()
	return Profile{
		Registry:  registry.NewMDNSRegistry(),
		Broker:    broker.NewHttpBroker(),
		Store:     store.NewFileStore(),
		Transport: transport.NewHTTPTransport(),
		Stream:    stream,
	}, err
}
