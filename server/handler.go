package server

import (
	"github.com/kynrai/go-micro/registry"
)

type Handler interface {
	Name() string
	Handler() interface{}
	Endpoints() []*registry.Endpoint
}

type Subscriber interface {
	Topic() string
	Subscriber() interface{}
	Endpoints() []*registry.Endpoint
}
