package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/cmd"
	example "github.com/micro/go-micro/examples/server/proto/example"
	"github.com/micro/go-micro/registry"
	"golang.org/x/net/context"
)

func init() {
	rand.Seed(time.Now().Unix())
}

// Built in random hashed node selector
type firstNodeSelector struct {
	opts registry.SelectorOptions
}

func (n *firstNodeSelector) Select(service string, opts ...registry.SelectOption) (registry.SelectNext, error) {
	services, err := n.opts.Registry.GetService(service)
	if err != nil {
		return nil, err
	}

	if len(services) == 0 {
		return nil, registry.ErrNotFound
	}

	var sopts registry.SelectOptions
	for _, opt := range opts {
		opt(&sopts)
	}

	for _, filter := range sopts.Filters {
		services = filter(services)
	}

	if len(services) == 0 {
		return nil, registry.ErrNotFound
	}

	if len(services[0].Nodes) == 0 {
		return nil, registry.ErrNotFound
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

// Return a new first node selector
func FirstNodeSelector(opts ...registry.SelectorOption) registry.Selector {
	var sopts registry.SelectorOptions
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

	fmt.Println("\n--- Call example ---\n")
	for i := 0; i < 10; i++ {
		call(i)
	}
}
