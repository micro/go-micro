// Package handler implements service debug handler embedded in go-micro services
package handler

import (
	"context"
	"time"

	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/debug/log"
	proto "github.com/asim/go-micro/v3/debug/proto"
	"github.com/asim/go-micro/v3/debug/stats"
	"github.com/asim/go-micro/v3/debug/trace"
	"github.com/asim/go-micro/v3/server"
)

// NewHandler returns an instance of the Debug Handler
func NewHandler(c client.Client) *Debug {
	return &Debug{
		log:   log.DefaultLog,
		stats: stats.DefaultStats,
		trace: trace.DefaultTracer,
	}
}

type Debug struct {
	// must honour the debug handler
	proto.DebugHandler
	// the logger for retrieving logs
	log log.Log
	// the stats collector
	stats stats.Stats
	// the tracer
	trace trace.Tracer
}

func (d *Debug) Health(ctx context.Context, req *proto.HealthRequest, rsp *proto.HealthResponse) error {
	rsp.Status = "ok"
	return nil
}

func (d *Debug) Stats(ctx context.Context, req *proto.StatsRequest, rsp *proto.StatsResponse) error {
	stats, err := d.stats.Read()
	if err != nil {
		return err
	}

	if len(stats) == 0 {
		return nil
	}

	// write the response values
	rsp.Timestamp = uint64(stats[0].Timestamp)
	rsp.Started = uint64(stats[0].Started)
	rsp.Uptime = uint64(stats[0].Uptime)
	rsp.Memory = stats[0].Memory
	rsp.Gc = stats[0].GC
	rsp.Threads = stats[0].Threads
	rsp.Requests = stats[0].Requests
	rsp.Errors = stats[0].Errors

	return nil
}

func (d *Debug) Trace(ctx context.Context, req *proto.TraceRequest, rsp *proto.TraceResponse) error {
	traces, err := d.trace.Read(trace.ReadTrace(req.Id))
	if err != nil {
		return err
	}

	for _, t := range traces {
		var typ proto.SpanType
		switch t.Type {
		case trace.SpanTypeRequestInbound:
			typ = proto.SpanType_INBOUND
		case trace.SpanTypeRequestOutbound:
			typ = proto.SpanType_OUTBOUND
		}
		rsp.Spans = append(rsp.Spans, &proto.Span{
			Trace:    t.Trace,
			Id:       t.Id,
			Parent:   t.Parent,
			Name:     t.Name,
			Started:  uint64(t.Started.UnixNano()),
			Duration: uint64(t.Duration.Nanoseconds()),
			Type:     typ,
			Metadata: t.Metadata,
		})
	}

	return nil
}

func (d *Debug) Log(ctx context.Context, stream server.Stream) error {
	req := new(proto.LogRequest)
	if err := stream.Recv(req); err != nil {
		return err
	}

	var options []log.ReadOption

	since := time.Unix(req.Since, 0)
	if !since.IsZero() {
		options = append(options, log.Since(since))
	}

	count := int(req.Count)
	if count > 0 {
		options = append(options, log.Count(count))
	}

	if req.Stream {
		// TODO: we need to figure out how to close the log stream
		// It seems like when a client disconnects,
		// the connection stays open until some timeout expires
		// or something like that; that means the map of streams
		// might end up leaking memory if not cleaned up properly
		lgStream, err := d.log.Stream()
		if err != nil {
			return err
		}
		defer lgStream.Stop()

		for record := range lgStream.Chan() {
			// copy metadata
			metadata := make(map[string]string)
			for k, v := range record.Metadata {
				metadata[k] = v
			}
			// send record
			if err := stream.Send(&proto.Record{
				Timestamp: record.Timestamp.Unix(),
				Message:   record.Message.(string),
				Metadata:  metadata,
			}); err != nil {
				return err
			}
		}

		// done streaming, return
		return nil
	}

	// get the log records
	records, err := d.log.Read(options...)
	if err != nil {
		return err
	}

	// send all the logs downstream
	for _, record := range records {
		// copy metadata
		metadata := make(map[string]string)
		for k, v := range record.Metadata {
			metadata[k] = v
		}
		// send record
		if err := stream.Send(&proto.Record{
			Timestamp: record.Timestamp.Unix(),
			Message:   record.Message.(string),
			Metadata:  metadata,
		}); err != nil {
			return err
		}
	}

	return nil
}
