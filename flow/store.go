package flow

import (
	"context"
)

type Store interface {
	Init() error
	Read(ctx context.Context, flow, rid string, key []byte) ([]byte, error)
	Write(ctx context.Context, flow, rid string, key []byte, val []byte) error
	Update(ctx context.Context, flow, rid string, key []byte, oldval []byte, newval []byte) error
	Clean(ctx context.Context, flow, rid string) error
	String() string
	Close(ctx context.Context) error
}
