package server

import (
	"github.com/alexapps/go-micro/server/debug"
)

func registerDebugHandler(s Server) {
	s.Handle(s.NewHandler(&debug.Debug{s.Options().DebugHandler}, InternalHandler(true)))
}
