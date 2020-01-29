package flow

import (
	"context"
	"errors"
)

var (
	// Not found error when flow is not found
	ErrFlowNotFound = errors.New("flow not found")
)

type FlowStore interface {
	Init() error
	Load(ctx context.Context, flow string) ([]*Step, error)
	Save(ctx context.Context, flow string, steps []*Step) error
	Modified(ctx context.Context, flow string) int64
	String() string
	Close(ctx context.Context) error
}

type DataStore interface {
	Init() error
	Read(ctx context.Context, flow, rid string, key []byte) ([]byte, error)
	Write(ctx context.Context, flow, rid string, key []byte, val []byte) error
	Update(ctx context.Context, flow, rid string, key []byte, oldval []byte, newval []byte) error
	Clean(ctx context.Context, flow, rid string) error
	String() string
	Close(ctx context.Context) error
}
