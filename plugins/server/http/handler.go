package http

import (
	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/server"
)

type httpHandler struct {
	opts server.HandlerOptions
	eps  []*registry.Endpoint
	hd   interface{}
}

func (h *httpHandler) Name() string {
	return "handler"
}

func (h *httpHandler) Handler() interface{} {
	return h.hd
}

func (h *httpHandler) Endpoints() []*registry.Endpoint {
	return h.eps
}

func (h *httpHandler) Options() server.HandlerOptions {
	return h.opts
}
