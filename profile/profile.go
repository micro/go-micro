// Package profileconfig provides grouped plugin profiles for go-micro
package profile

import (
	"os"
	"strings"

	natslib "github.com/nats-io/nats.go"
	"go-micro.dev/v5/broker"
	"go-micro.dev/v5/broker/nats"
	"go-micro.dev/v5/registry"
	nreg "go-micro.dev/v5/registry/nats"
	"go-micro.dev/v5/store"
	nstore "go-micro.dev/v5/store/nats-js-kv"

	"go-micro.dev/v5/transport"
	ntx "go-micro.dev/v5/transport/nats"
)

type Profile struct {
	Registry  registry.Registry
	Broker    broker.Broker
	Store     store.Store
	Transport transport.Transport
}

// LocalProfile returns a profile with local mDNS as the registry, HTTP as the broker, file as the store, and HTTP as the transport
// It is used for local development and testing
func LocalProfile() Profile {
	return Profile{
		Registry:  registry.NewMDNSRegistry(),
		Broker:    broker.NewHttpBroker(),
		Store:     store.NewFileStore(),
		Transport: transport.NewHTTPTransport(),
	}
}

// NatsProfile returns a profile with NATS as the registry, broker, store, and transport
// It uses the environment variable MICR_NATS_ADDRESS to set the NATS server address
// If the variable is not set, it defaults to nats://0.0.0.0:4222 which will connect to a local NATS server
func NatsProfile() Profile {
	addr := os.Getenv("MICRO_NATS_ADDRESS")
	if addr == "" {
		addr = "nats://0.0.0.0:4222"
	}
	// Split the address by comma, trim whitespace, and convert to a slice of strings
	addrs := splitNatsAdressList(addr)
	reg := nreg.NewNatsRegistry(registry.Addrs(addrs...))
	brok := nats.NewNatsBroker(broker.Addrs(addrs...))
	st := nstore.NewStore(nstore.NatsOptions(natslib.Options{Servers: addrs}))
	tx := ntx.NewTransport(ntx.Options(natslib.Options{Servers: addrs}))

	registry.DefaultRegistry = reg
	broker.DefaultBroker = brok
	store.DefaultStore = st
	transport.DefaultTransport = tx
	return Profile{
		Registry:  reg,
		Broker:    brok,
		Store:     st,
		Transport: tx,
	}
}

func splitNatsAdressList(addr string) []string {
	// Split the address by comma
	addrs := strings.Split(addr, ",")
	// Trim any whitespace from each address
	for i, a := range addrs {
		addrs[i] = strings.TrimSpace(a)
	}
	return addrs
}

// Add more profiles as needed, e.g. grpc
