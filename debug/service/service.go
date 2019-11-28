package service

import (
	"context"
	"fmt"
	"time"

	"github.com/micro/go-micro/client"

	"github.com/micro/go-micro/debug/log"
	pb "github.com/micro/go-micro/debug/proto"
)

// Debug provides debug service client
type Debug struct {
	dbg pb.DebugService
}

// NewDebug provides Debug service implementation
func NewDebug(name string) *Debug {
	// create default client
	cli := client.DefaultClient

	return &Debug{
		dbg: pb.NewDebugService(name, cli),
	}
}

// Logs queries the service logs and returns a channel to read the logs from
func (d *Debug) Logs(opts ...log.ReadOption) (<-chan log.Record, error) {
	options := log.ReadOptions{}
	// initialize the read options
	for _, o := range opts {
		o(&options)
	}

	req := &pb.LogRequest{}
	if !options.Since.IsZero() {
		req.Since = options.Since.UnixNano()
	}

	if options.Count > 0 {
		req.Count = int64(options.Count)
	}

	// get the log stream
	stream, err := d.dbg.Logs(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed getting log stream: %s", err)
	}

	// log channel for streaming logs
	logChan := make(chan log.Record)
	// go stream logs
	go d.streamLogs(logChan, stream)

	return logChan, nil
}

func (d *Debug) streamLogs(logChan chan log.Record, stream pb.Debug_LogsService) {
	defer stream.Close()

	for {
		resp, err := stream.Recv()
		if err != nil {
			break
		}

		metadata := make(map[string]string)
		for k, v := range resp.Metadata {
			metadata[k] = v
		}

		record := log.Record{
			Timestamp: time.Unix(0, resp.Timestamp),
			Value:     resp.Value,
			Metadata:  metadata,
		}

		logChan <- record
	}

	close(logChan)
}
