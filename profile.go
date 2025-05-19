// Package profileconfig provides grouped plugin profiles for go-micro
package profileconfig

import (
	"os"
	"go-micro.dev/v5/broker"
	"go-micro.dev/v5/broker/http"
	"go-micro.dev/v5/broker/nats"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/registry/mdns"
	"go-micro.dev/v5/registry/nats"
	"go-micro.dev/v5/store"
	"go-micro.dev/v5/store/file"
	"go-micro.dev/v5/transport"
	"go-micro.dev/v5/transport/http"
)

type ProfileConfig struct {
	Registry  registry.Registry
	Broker    broker.Broker
	Store     store.Store
	Transport transport.Transport
}

func LocalProfile() ProfileConfig {
	return ProfileConfig{
		Registry:  mdns.NewRegistry(),
		Broker:    http.NewBroker(),
		Store:     file.NewStore(),
		Transport: http.NewTransport(),
	}
}

func NatsProfile() ProfileConfig {
	addr := os.Getenv("MICRO_NATS_ADDRESS")
	return ProfileConfig{
		Registry:  nats.NewRegistry(registry.Addrs(addr)),
		Broker:    nats.NewBroker(broker.Addrs(addr)),
		Store:     file.NewStore(), // or nats-backed store if available
		Transport: http.NewTransport(), // or nats transport if available
	}
}

// Add more profiles as needed, e.g. grpc
