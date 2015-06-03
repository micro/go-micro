package server

import (
	"github.com/myodc/go-micro/registry"
)

type Handler interface {
	Name() string
	Handler() interface{}
	Endpoints() []*registry.Endpoint
}
