// Package mux provides proxy muxing
package mux

import (
	"context"
	"sync"

	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/debug/handler"
	"github.com/asim/go-micro/v3/network/proxy"
	"github.com/asim/go-micro/v3/server"
)

// Server is a proxy muxer that incudes the use of the DefaultHandler
type Server struct {
	// name of service
	Name string
	// Proxy handler
	Proxy proxy.Proxy
}

var (
	once sync.Once
)

func (s *Server) ProcessMessage(ctx context.Context, msg server.Message) error {
	if msg.Topic() == s.Name {
		return server.DefaultRouter.ProcessMessage(ctx, msg)
	}
	return s.Proxy.ProcessMessage(ctx, msg)
}

func (s *Server) ServeRequest(ctx context.Context, req server.Request, rsp server.Response) error {
	if req.Service() == s.Name {
		return server.DefaultRouter.ServeRequest(ctx, req, rsp)
	}
	return s.Proxy.ServeRequest(ctx, req, rsp)
}

func New(name string, p proxy.Proxy) *Server {
	// only register this once
	once.Do(func() {
		server.DefaultRouter.Handle(
			// inject the debug handler
			server.DefaultRouter.NewHandler(
				handler.NewHandler(client.DefaultClient),
				server.InternalHandler(true),
			),
		)
	})

	return &Server{
		Name:  name,
		Proxy: p,
	}
}
