package client

import (
	"context"

	"go-micro.dev/v4/registry"
)

// CallFunc represents the individual call func
type CallFunc func(ctx context.Context, node *registry.Node, req Request, rsp interface{}, opts CallOptions) error

// CallWrapper is a low level wrapper for the CallFunc
type CallWrapper func(CallFunc) CallFunc

// PublishFunc represents the individual publish func
type PublishFunc func(ctx context.Context, msg Message, opts ...PublishOption) error

// CallWrapper is a low level wrapper for the PublishFunc
type PublishWrapper func(PublishFunc) PublishFunc

// Wrapper wraps a client and returns a client
type Wrapper func(Client) Client

// StreamWrapper wraps a Stream and returns the equivalent
type StreamWrapper func(Stream) Stream
