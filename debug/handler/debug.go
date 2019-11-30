// Pacjage handler implements service debug handler
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
	var options []log.ReadOption

	since := time.Unix(0, req.Since)
	if !since.IsZero() {
		options = append(options, log.Since(since))
	}

	count := int(req.Count)
	if count > 0 {
		options = append(options, log.Count(count))
	}

	if req.Stream {
		stop := make(chan bool)
		defer close(stop)

		// TODO: figure out how to close log stream
		// It seems when the client disconnects,
		// the connection stays open until some timeout expires
		// or something like that; that means the map of streams
		// might end up bloating if not cleaned up properly
		records := d.log.Stream(stop)
		for record := range records {
			if err := d.sendRecord(record, stream); err != nil {
				return err
			}
		}
		// done streaming, return
		return nil
	}

	// get the log records
	records := d.log.Read(options...)
	// send all the logs downstream
	for _, record := range records {
		if err := d.sendRecord(record, stream); err != nil {
			return err
		}
	}

	return nil
}

func (d *Debug) sendRecord(record log.Record, stream proto.Debug_LogsStream) error {
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

	return nil
}
