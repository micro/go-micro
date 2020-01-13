package flow

import "context"

type StateStore interface {
	Init() error
	Read(ctx context.Context, flow, rid string, key []byte) ([]byte, error)
	Write(ctx context.Context, flow, rid string, key []byte, val []byte) error
	Update(ctx context.Context, flow, rid string, key []byte, oldval []byte, newval []byte) error
	Clean(ctx context.Context, flow, rid string) error
	String() string
	Close(ctx context.Context) error
}

type DataStore interface {
	Init() error
	Read(ctx context.Context, flow, rid string, key []byte) ([]byte, error)
	Write(ctx context.Context, flow, rid string, key []byte, val []byte) error
	Delete(ctx context.Context, flow, rid string, key []byte) error
	Clean(ctx context.Context, flow, rid string) error
	String() string
	Close(ctx context.Context) error
}

type FlowStore interface {
	Init() error
	Write(ctx context.Context, flow string, data []byte) error
	Read(ctx context.Context, flow string) ([]byte, error)
	String() string
	Close(ctx context.Context) error
}
