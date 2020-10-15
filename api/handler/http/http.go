// Package http is a http reverse proxy handler
package http

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/micro/go-micro/v3/api"
	"github.com/micro/go-micro/v3/api/handler"
	"github.com/micro/go-micro/v3/registry"
)

const (
	Handler = "http"
)

type httpHandler struct {
	options handler.Options

	// set with different initializer
	s *api.Service
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	service, err := h.getService(r)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	if len(service) == 0 {
		w.WriteHeader(404)
		return
	}

	rp, err := url.Parse(service)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	httputil.NewSingleHostReverseProxy(rp).ServeHTTP(w, r)
}

// getService returns the service for this request from the selector
func (h *httpHandler) getService(r *http.Request) (string, error) {
	var service *api.Service

	if h.s != nil {
		// we were given the service
		service = h.s
	} else if h.options.Router != nil {
		// try get service from router
		s, err := h.options.Router.Route(r)
		if err != nil {
			return "", err
		}
		service = s
	} else {
		// we have no way of routing the request
		return "", errors.New("no route found")
	}

	// get the nodes for this service
	var nodes []*registry.Node
	for _, srv := range service.Services {
		nodes = append(nodes, srv.Nodes...)
	}

	// select a random node
	if len(nodes) == 0 {
		return "", errors.New("no route found")
	}
	node := nodes[rand.Int()%len(nodes)]

	return fmt.Sprintf("http://%s", node.Address), nil
}

func (h *httpHandler) String() string {
	return "http"
}

// NewHandler returns a http proxy handler
func NewHandler(opts ...handler.Option) handler.Handler {
	options := handler.NewOptions(opts...)

	return &httpHandler{
		options: options,
	}
}

// WithService creates a handler with a service
func WithService(s *api.Service, opts ...handler.Option) handler.Handler {
	options := handler.NewOptions(opts...)

	return &httpHandler{
		options: options,
		s:       s,
	}
}
