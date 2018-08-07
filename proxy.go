package micro

import (
	"github.com/micro/go-micro/registry/consul"
	"github.com/micro/go-micro/transport"
	"github.com/micro/go-micro/transport/http"
)

type proxy struct{}

func newProxy(opts ...Option) Proxy {
	return &proxy{}
}

func (p *proxy) Service(opts ...Option) Service {
	return newService(opts...)
}

func (p *proxy) String() string {
	return "micro"
}
