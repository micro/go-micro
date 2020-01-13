package flow

import (
	"context"
)

type EventHandler interface {
	Init() error
	FlowStart(ctx context.Context, flow, rid string) error
	FlowEnd(ctx context.Context, flow, rid string) error
	FlowFail(ctx context.Context, flow, rid string, err error) error
	NodeStart(ctx context.Context, flow, rid, node string) error
	NodeEnd(ctx context.Context, flow, rid, node string) error
	NodeFail(ctx context.Context, flow, rid, node string, err error) error
	OperationStart(ctx context.Context, flow, rid, node, operation string) error
	OperationEnd(ctx context.Context, flow, rid, node, operation string) error
	OperationFail(ctx context.Context, flow, rid, node, operation string, err error) error
	Close() error
}
