// Package grpc provides a grpc server with features; acme, cors, etc
package grpc

import (
	"net/http"

	api "github.com/micro/go-micro/v2/api/server"
	"github.com/micro/go-micro/v2/server"
	gRPC "github.com/micro/go-micro/v2/server/grpc"
)

// Server serves API requests
type Server struct {
	grpc server.Server
}

// NewServer returns an gRPC server
func NewServer(address string) api.Server {
	return &Server{
		grpc: gRPC.NewServer(server.Address(address)),
	}
}

// Address of the server
func (s *Server) Address() string {
	return s.grpc.Options().Address
}

// Init the gPRC server
func (s *Server) Init(opts ...api.Option) error {
	options := api.Options{}
	for _, o := range opts {
		o(&options)
	}

	return s.grpc.Init(
		server.EnableACME(options.EnableACME),
		server.ACMEProvider(options.ACMEProvider),
		server.EnableTLS(options.EnableTLS),
		server.ACMEHosts(options.ACMEHosts...),
		server.TLSConfig(options.TLSConfig),
	)
}

// Handle a request
func (s *Server) Handle(path string, handler http.Handler) {
	s.grpc.Handle(server.NewHandler(handler.ServeHTTP))
}

// Start the server
func (s *Server) Start() error {
	return s.grpc.Start()
}

// Stop the server
func (s *Server) Stop() error {
	return s.grpc.Stop()
}
