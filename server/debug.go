package server

import (
	"github.com/micro/go-micro/server/debug"
)

// We use this to wrap any debug handlers so we preserve the signature Debug.{Method}
type Debug struct {
	debug.DebugHandler
}

func registerDebugHandler(s Server) {
	s.Handle(s.NewHandler(&Debug{s.Options().DebugHandler}, InternalHandler(true)))
}
