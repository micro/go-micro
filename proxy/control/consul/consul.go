// Package consul provides Consul Connect control plane
package consul

import (
	"log"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/connect"
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/registry/consul"
	"github.com/micro/go-micro/transport"
)

type proxyService struct {
	c *connect.Service
	micro.Service
}

func newService(opts ...micro.Option) micro.Service {
	// we need to use the consul registry to register connect applications
	r := consul.NewRegistry(
		consul.Connect(),
	)

	// pass in the registry as part of our options
	newOpts := append([]micro.Option{micro.Registry(r)}, opts...)

	// service := micro.NewService(newOpts...)
	service := micro.NewService(newOpts...)

	// get the consul address
	addrs := service.Server().Options().Registry.Options().Addrs

	// set the config
	config := api.DefaultConfig()
	if len(addrs) > 0 {
		config.Address = addrs[0]
	}

	// create consul client
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	// create connect service using the service name
	svc, err := connect.NewService(service.Server().Options().Name, client)
	if err != nil {
		log.Fatal(err)
	}

	// setup transport tls config
	service.Options().Transport.Init(
		transport.TLSConfig(svc.ServerTLSConfig()),
	)

	// setup broker tls config
	service.Options().Broker.Init(
		broker.TLSConfig(svc.ServerTLSConfig()),
	)

	// return a new proxy enabled service
	return &proxyService{
		c:       svc,
		Service: service,
	}
}

func (p *proxyService) String() string {
	return "consul"
}

// NewService returns a Consul Connect-Native micro.Service
func NewService(opts ...micro.Option) micro.Service {
	return newService(opts...)
}
