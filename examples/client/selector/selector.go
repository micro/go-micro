package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	example "github.com/asim/go-micro/examples/v3/server/proto/example"
	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/selector"
	"github.com/asim/go-micro/v3/cmd"
	"github.com/asim/go-micro/v3/registry"
)

func init() {
	rand.Seed(time.Now().Unix())
}

// Built in random hashed node selector
type firstNodeSelector struct {
	opts selector.Options
}

func (n *firstNodeSelector) Init(opts ...selector.Option) error {
	for _, o := range opts {
		o(&n.opts)
	}
	return nil
}

func (n *firstNodeSelector) Options() selector.Options {
	return n.opts
}

func (n *firstNodeSelector) Select(service string, opts ...selector.SelectOption) (selector.Next, error) {
	services, err := n.opts.Registry.GetService(service)
	if err != nil {
		return nil, err
	}

	if len(services) == 0 {
		return nil, selector.ErrNotFound
	}

	var sopts selector.SelectOptions
	for _, opt := range opts {
		opt(&sopts)
	}

	for _, filter := range sopts.Filters {
		services = filter(services)
	}

	if len(services) == 0 {
		return nil, selector.ErrNotFound
	}

	if len(services[0].Nodes) == 0 {
		return nil, selector.ErrNotFound
	}

	return func() (*registry.Node, error) {
		return services[0].Nodes[0], nil
	}, nil
}

func (n *firstNodeSelector) Mark(service string, node *registry.Node, err error) {
	return
}

func (n *firstNodeSelector) Reset(service string) {
	return
}

func (n *firstNodeSelector) Close() error {
	return nil
}

func (n *firstNodeSelector) String() string {
	return "first"
}

// Return a new first node selector
func FirstNodeSelector(opts ...selector.Option) selector.Selector {
	var sopts selector.Options
	for _, opt := range opts {
		opt(&sopts)
	}
	if sopts.Registry == nil {
		sopts.Registry = registry.DefaultRegistry
	}
	return &firstNodeSelector{sopts}
}

func call(i int) {
	// Create new request to service go.micro.srv.example, method Example.Call
	req := client.NewRequest("go.micro.srv.example", "Example.Call", &example.Request{
		Name: "John",
	})

	rsp := &example.Response{}

	// Call service
	if err := client.Call(context.Background(), req, rsp); err != nil {
		fmt.Println("call err: ", err, rsp)
		return
	}

	fmt.Println("Call:", i, "rsp:", rsp.Msg)
}

func main() {
	cmd.Init()

	client.DefaultClient = client.NewClient(
		client.Selector(FirstNodeSelector()),
	)

	fmt.Println("\n--- Call example ---")
	for i := 0; i < 10; i++ {
		call(i)
	}
}
