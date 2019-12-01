// Package mux provides proxy muxing
package mux

import (
	"context"
	"sync"

	proto "github.com/micro/go-micro/debug/proto"
	"github.com/micro/go-micro/debug/handler"
	"github.com/micro/go-micro/proxy"
	"github.com/micro/go-micro/server"
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
		proto.RegisterDebugHandler(
			server.DefaultServer,
			handler.DefaultHandler,
			server.InternalHandler(true),
		)
	})

	return &Server{
		Name:  name,
		Proxy: p,
	}
}
