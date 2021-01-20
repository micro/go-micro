// Package shard implements the sharding call strategy
package shard

import (
	"hash/crc32"

	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/selector"
	"github.com/asim/go-micro/v3/metadata"
	"github.com/asim/go-micro/v3/registry"

	"context"
)

type shard struct {
	key string
	client.Client
}

func (s *shard) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	// get headers
	md, ok := metadata.FromContext(ctx)
	if !ok {
		// noop, defer to client
		return s.Client.Call(ctx, req, rsp, opts...)
	}

	// get key val
	val := md[s.key]

	// noop on nil value
	if len(val) == 0 {
		return s.Client.Call(ctx, req, rsp, opts...)
	}

	// checksum it
	cs := crc32.ChecksumIEEE([]byte(val))

	nOpts := append(opts, client.WithSelectOption(
		// create a selector strategy
		selector.WithStrategy(func(services []*registry.Service) selector.Next {
			// flatten
			var nodes []*registry.Node
			for _, service := range services {
				nodes = append(nodes, service.Nodes...)
			}

			// create the next func that always returns our node
			return func() (*registry.Node, error) {
				if len(nodes) == 0 {
					return nil, selector.ErrNoneAvailable
				}
				return nodes[cs%uint32(len(nodes))], nil
			}
		}),
	))

	return s.Client.Call(ctx, req, rsp, nOpts...)
}

// NewClientWrapper is a wrapper which shards based on a header key value
func NewClientWrapper(key string) client.Wrapper {
	return func(c client.Client) client.Client {
		return &shard{
			key:    key,
			Client: c,
		}
	}
}
