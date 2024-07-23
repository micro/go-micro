package test

import (
	"testing"

	"go-micro.dev/v5"
	"go-micro.dev/v5/broker"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/server"
	"go-micro.dev/v5/transport"
	"go-micro.dev/v5/util/test"
)

func BenchmarkService(b *testing.B) {
	cfg := ServiceTestConfig{
		Name:       "test-service",
		NewService: newService,
		Parallel:   []int{1, 8, 16, 32, 64},
		Sequential: []int{0},
		Streams:    []int{0},
		//		PubSub:     []int{10},
	}

	cfg.Run(b)
}

func newService(name string, opts ...micro.Option) (micro.Service, error) {
	r := registry.NewMemoryRegistry(
		registry.Services(test.Data),
	)

	b := broker.NewMemoryBroker()

	t := transport.NewHTTPTransport()
	c := client.NewClient(
		client.Transport(t),
		client.Broker(b),
	)

	s := server.NewRPCServer(
		server.Name(name),
		server.Registry(r),
		server.Transport(t),
		server.Broker(b),
	)

	if err := s.Init(); err != nil {
		return nil, err
	}

	options := []micro.Option{
		micro.Name(name),
		micro.Server(s),
		micro.Client(c),
		micro.Registry(r),
		micro.Broker(b),
	}
	options = append(options, opts...)

	srv := micro.NewService(options...)

	return srv, nil
}
