package handler

import (
	"context"
	"runtime"
	"time"

	"github.com/micro/go-micro/debug/log"

	proto "github.com/micro/go-micro/debug/proto"
)

var (
	// DefaultHandler is default debug handler
	DefaultHandler = newDebug()
)

type Debug struct {
	started int64
	proto.DebugHandler
	log log.Log
}

func newDebug() *Debug {
	return &Debug{
		started: time.Now().Unix(),
		log:     log.DefaultLog,
	}
}

func (d *Debug) Health(ctx context.Context, req *proto.HealthRequest, rsp *proto.HealthResponse) error {
	rsp.Status = "ok"
	return nil
}

func (d *Debug) Stats(ctx context.Context, req *proto.StatsRequest, rsp *proto.StatsResponse) error {
	var mstat runtime.MemStats
	runtime.ReadMemStats(&mstat)

	rsp.Started = uint64(d.started)
	rsp.Uptime = uint64(time.Now().Unix() - d.started)
	rsp.Memory = mstat.Alloc
	rsp.Gc = mstat.PauseTotalNs
	rsp.Threads = uint64(runtime.NumGoroutine())
	return nil
}

func (d *Debug) Logs(ctx context.Context, req *proto.LogRequest, stream proto.Debug_LogsStream) error {
	var records []log.Record
	since := time.Unix(0, req.Since)
	if !since.IsZero() {
		records = d.log.Read(log.Since(since))
	} else {
		records = d.log.Read(log.Count(int(req.Count)))
	}

	// TODO: figure out the stream later on
	for _, record := range records {
		metadata := make(map[string]string)
		for k, v := range record.Metadata {
			metadata[k] = v
		}

		recLog := &proto.Log{
			Timestamp: record.Timestamp.UnixNano(),
			Value:     record.Value.(string),
			Metadata:  metadata,
		}

		if err := stream.Send(recLog); err != nil {
			return err
		}
	}

	return nil
}
