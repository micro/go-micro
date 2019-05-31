// Package rpc provides an rpc handler
package rpc

import (
	"context"

	proto "github.com/micro/go-micro/debug/handler/rpc/proto"
)

type Debug struct {
}

func (d *Debug) Health(ctx context.Context, req *proto.Request, rsp *proto.HealthResponse) error {
	return nil
}

func (d *Debug) Log(ctx context.Context, req *proto.Request, rsp *proto.LogResponse) error {
	return nil
}

func (d *Debug) Stats(ctx context.Context, req *proto.Request, rsp *proto.StatsResponse) error {
	return nil
}

func (d *Debug) Trace(ctx context.Context, req *proto.Request, rsp *proto.TraceResponse) error {
	return nil
}
