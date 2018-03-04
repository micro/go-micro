package debug

import (
	"context"
	"runtime"
	"time"

	proto "github.com/micro/go-micro/server/debug/proto"
)

// The debug handler represents an internal server handler
// used to determine health, status and env info about
// a service node. It's akin to Google's /statusz, /healthz,
// and /varz
type DebugHandler interface {
	Health(ctx context.Context, req *proto.HealthRequest, rsp *proto.HealthResponse) error
	Stats(ctx context.Context, req *proto.StatsRequest, rsp *proto.StatsResponse) error
}

// Our own internal handler
type debug struct {
	started int64
}

var (
	DefaultDebugHandler DebugHandler = newDebug()
)

func newDebug() *debug {
	return &debug{
		started: time.Now().Unix(),
	}
}

func (d *debug) Health(ctx context.Context, req *proto.HealthRequest, rsp *proto.HealthResponse) error {
	rsp.Status = "ok"
	return nil
}

func (d *debug) Stats(ctx context.Context, req *proto.StatsRequest, rsp *proto.StatsResponse) error {
	var mstat runtime.MemStats
	runtime.ReadMemStats(&mstat)

	rsp.Started = uint64(d.started)
	rsp.Uptime = uint64(time.Now().Unix() - d.started)
	rsp.Memory = mstat.Alloc
	rsp.Gc = mstat.PauseTotalNs
	rsp.Threads = uint64(runtime.NumGoroutine())
	return nil
}
