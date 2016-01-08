package server

import (
	"github.com/micro/go-micro/server/debug"
	proto "github.com/micro/go-micro/server/debug/proto"

	"golang.org/x/net/context"
)

// We use this to wrap any debug handlers so we preserve the signature Debug.{Method}
type Debug struct {
	debug.DebugHandler
}

func (d *Debug) Health(ctx context.Context, req *proto.HealthRequest, rsp *proto.HealthResponse) error {
	return d.DebugHandler.Health(ctx, req, rsp)
}

func registerDebugHandler(s Server) {
	s.Handle(s.NewHandler(&Debug{s.Options().DebugHandler}, InternalHandler(true)))
}
