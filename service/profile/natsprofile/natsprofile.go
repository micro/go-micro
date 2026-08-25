// Package natsprofile provides the NATS-backed plugin profile.
//
// It lives outside service/profile so that binaries which never select the
// nats profile do not link the NATS registry, broker, store, transport, and
// events machinery. Importing this package (directly, or via
// go-micro.dev/v6/cmd/defaults) registers the profile under the name "nats".
package natsprofile

import (
	"os"
	"strings"

	natslib "github.com/nats-io/nats.go"
	"go-micro.dev/v6/broker"
	"go-micro.dev/v6/broker/nats"
	nevents "go-micro.dev/v6/events/natsjs"
	"go-micro.dev/v6/registry"
	nreg "go-micro.dev/v6/registry/nats"
	"go-micro.dev/v6/service/profile"
	"go-micro.dev/v6/store"
	nstore "go-micro.dev/v6/store/nats-js-kv"
	"go-micro.dev/v6/transport"
	ntx "go-micro.dev/v6/transport/nats"
)

func init() {
	profile.Register("nats", NatsProfile)
}

// NatsProfile returns a profile with NATS as the registry, broker, store, and transport
// It uses the environment variable MICRO_NATS_ADDRESS to set the NATS server address
// If the variable is not set, it defaults to nats://0.0.0.0:4222 which will connect to a local NATS server
func NatsProfile() (profile.Profile, error) {
	addr := os.Getenv("MICRO_NATS_ADDRESS")
	if addr == "" {
		addr = "nats://0.0.0.0:4222"
	}
	// Split the address by comma, trim whitespace, and convert to a slice of strings
	addrs := splitNatsAddressList(addr)

	reg := nreg.NewNatsRegistry(registry.Addrs(addrs...))

	nopts := natslib.GetDefaultOptions()
	nopts.Servers = addrs
	brok := nats.NewNatsBroker(broker.Addrs(addrs...), nats.Options(nopts))

	st := nstore.NewStore(nstore.NatsOptions(natslib.Options{Servers: addrs}))
	tx := ntx.NewTransport(ntx.Options(natslib.Options{Servers: addrs}))

	stream, err := nevents.NewStream(
		nevents.Address(addr),
	)

	registry.DefaultRegistry = reg
	broker.DefaultBroker = brok
	store.DefaultStore = st
	transport.DefaultTransport = tx
	return profile.Profile{
		Registry:  reg,
		Broker:    brok,
		Store:     st,
		Transport: tx,
		Stream:    stream,
	}, err
}

func splitNatsAddressList(addr string) []string {
	// Split the address by comma
	addrs := strings.Split(addr, ",")
	// Trim any whitespace from each address
	for i, a := range addrs {
		addrs[i] = strings.TrimSpace(a)
	}
	return addrs
}
