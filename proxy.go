package micro

import (
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/connect"
	"github.com/micro/go-micro/registry/consul"
	"github.com/micro/go-micro/transport"
)

type proxy struct{}

type proxyService struct {
	Service
}

func newProxy(opts ...Option) Proxy {
	return &proxy{}
}

func (p *proxy) Service(opts ...Option) Service {
	r := consul.NewRegistry(
		consul.Connect(),
	)
	newOpts := append([]Option{Registry(r)}, opts...)
	return &proxyService{newService(newOpts...)}
}

func (p *proxy) String() string {
	return "micro"
}

func (p *proxyService) Run() error {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return err
	}

	// create connect service
	svc, err := connect.NewService(p.Service.Server().Options().Name, client)
	if err != nil {
		return err
	}
	defer svc.Close()

	// setup connect tls config
	p.Service.Options().Transport.Init(
		transport.TLSConfig(svc.ServerTLSConfig()),
	)

	// run service
	return p.Service.Run()
}
