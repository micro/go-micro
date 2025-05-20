// Package profileconfig provides grouped plugin profiles for go-micro
package profile

import (
	"os"

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

func LocalProfile() Profile {
	return Profile{
		Registry:  registry.NewMDNSRegistry(),
		Broker:    broker.NewHttpBroker(),
		Store:     store.NewFileStore(),
		Transport: transport.NewHTTPTransport(),
	}
}

func NatsProfile() Profile {
	addr := os.Getenv("MICRO_NATS_ADDRESS")
	reg := nreg.NewNatsRegistry(registry.Addrs(addr))
	brok := nats.NewNatsBroker(broker.Addrs(addr))
	// this might be wrong, look for a better way to set this up
	st := nstore.NewStore(nstore.NatsOptions(natslib.Options{Url: addr}))
	// same, double check for single url vs slice of Server
	tx := ntx.NewTransport(ntx.Options(natslib.Options{Url: addr}))

	registry.DefaultRegistry = reg
	broker.DefaultBroker = brok
	store.DefaultStore = st
	transport.DefaultTransport = tx
	return Profile{
		Registry: reg,
		Broker:   brok,
		// this might be wrong, look for a better way to set this up
		Store: st,
		// same, double check for single url vs slice of Server
		Transport: tx,
	}
}

// Add more profiles as needed, e.g. grpc
