// Package profileconfig provides grouped plugin profiles for go-micro
package profile

import (
	"os"
	"go-micro.dev/v5/broker"
	"go-micro.dev/v5/broker/http"
	"go-micro.dev/v5/broker/nats"
	"go-micro.dev/v5/registry"
	nreg "go-micro.dev/v5/registry/nats"
	"go-micro.dev/v5/store"

	"go-micro.dev/v5/transport"

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
		Broker:    http.NewHttpBroker(),
		Store:     store.NewFileStore(),
		Transport: transport.NewHTTPTransport(),
	}
}

func NatsProfile() Profile {
	addr := os.Getenv("MICRO_NATS_ADDRESS")
	return Profile{
		Registry:  nreg.NewNatsRegistry(registry.Addrs(addr)),
		Broker:    nats.NewNatsBroker(broker.Addrs(addr)),
		Store:     store.NewFileStore(), // or nats-backed store if available
		Transport: transport.NewHTTPTransport(), // or nats transport if available
	}
}

// Add more profiles as needed, e.g. grpc
