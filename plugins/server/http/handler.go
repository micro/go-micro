package http

import (
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/server"
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
