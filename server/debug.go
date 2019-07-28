package server

import (
	"github.com/micro/go-micro/server/debug"
)

func registerDebugHandler(s Server) {
	s.Handle(s.NewHandler(&debug.Debug{s.Options().DebugHandler}, InternalHandler(true)))
}
