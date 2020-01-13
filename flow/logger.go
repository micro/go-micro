package flow

import (
	"context"
)

type Logger interface {
	Init() error
	Log(ctx context.Context, flow, rid, msg string) error
}
