// Package roundrobin implements a roundrobin call strategy
package roundrobin

import (
	"sync"

	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/selector"
	"github.com/asim/go-micro/v3/registry"

	"context"
)

type roundrobin struct {
	sync.Mutex
	rr map[string]int
	client.Client
}

func (s *roundrobin) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
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
				s.Lock()
				// get counter
				rr := s.rr[req.Service()]
				// get node
				node := nodes[rr%len(nodes)]
				// increment
				rr++
				// save
				s.rr[req.Service()] = rr
				s.Unlock()

				return node, nil
			}
		}),
	))

	return s.Client.Call(ctx, req, rsp, nOpts...)
}

// NewClientWrapper is a wrapper which roundrobins requests
func NewClientWrapper() client.Wrapper {
	return func(c client.Client) client.Client {
		return &roundrobin{
			rr:     make(map[string]int),
			Client: c,
		}
	}
}
