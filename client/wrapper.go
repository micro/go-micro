package client

import (
	"context"

	"github.com/micro/go-micro/v2/registry"
)

// CallFunc represents the individual call func
type CallFunc func(ctx context.Context, node *registry.Node, req Request, rsp interface{}, opts CallOptions) error

// CallWrapper is a low level wrapper for the CallFunc
type CallWrapper func(CallFunc) CallFunc

// Wrapper wraps a client and returns a client
type Wrapper func(Client) Client

// StreamWrapper wraps a Stream and returns the equivalent
type StreamWrapper func(Stream) Stream
