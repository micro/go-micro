package debug

import (
	proto "github.com/micro/go-micro/server/debug/proto"

	"golang.org/x/net/context"
)

// The debug handler represents an internal server handler
// used to determine health, status and env info about
// a service node. It's akin to Google's /statusz, /healthz,
// and /varz
type DebugHandler interface {
	Health(ctx context.Context, req *proto.HealthRequest, rsp *proto.HealthResponse) error
}

// Our own internal handler
type debug struct{}

var (
	DefaultDebugHandler = new(debug)
)

func (d *debug) Health(ctx context.Context, req *proto.HealthRequest, rsp *proto.HealthResponse) error {
	rsp.Status = "ok"
	return nil
}
