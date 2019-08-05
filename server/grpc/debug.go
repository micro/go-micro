package grpc

import (
	"github.com/micro/go-micro/server"
	"github.com/micro/go-micro/server/debug"
)

func registerDebugHandler(s server.Server) {
	s.Handle(s.NewHandler(&debug.Debug{s.Options().DebugHandler}, server.InternalHandler(true)))
}
