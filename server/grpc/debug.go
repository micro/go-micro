package grpc

import (
	"github.com/alexapps/go-micro/server"
	"github.com/alexapps/go-micro/server/debug"
)

func registerDebugHandler(s server.Server) {
	s.Handle(s.NewHandler(&debug.Debug{s.Options().DebugHandler}, server.InternalHandler(true)))
}
