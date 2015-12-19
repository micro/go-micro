package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/cmd"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/selector"
	"golang.org/x/net/context"

	example "github.com/micro/go-micro/examples/server/proto/example"
)

// Built in random hashed node selector
type dcSelector struct {
	opts selector.Options
}

var (
	datacenter = "local"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func (n *dcSelector) Select(service string, opts ...selector.SelectOption) (selector.Next, error) {
	services, err := n.opts.Registry.GetService(service)
	if err != nil {
		return nil, err
	}

	if len(services) == 0 {
		return nil, selector.ErrNotFound
	}

	var nodes []*registry.Node

	// Filter the nodes for datacenter
	for _, service := range services {
		for _, node := range service.Nodes {
			if node.Metadata["datacenter"] == datacenter {
				nodes = append(nodes, node)
			}
		}
	}

	if len(nodes) == 0 {
		return nil, selector.ErrNotFound
	}

	var i int
	var mtx sync.Mutex

	return func() (*registry.Node, error) {
		mtx.Lock()
		defer mtx.Unlock()
		i++
		return nodes[i%len(nodes)], nil
	}, nil
}

func (n *dcSelector) Mark(service string, node *registry.Node, err error) {
	return
}

func (n *dcSelector) Reset(service string) {
	return
}

func (n *dcSelector) Close() error {
	return nil
}

func (n *dcSelector) String() string {
	return "dc"
}

// Return a new first node selector
func DCSelector(opts ...selector.Option) selector.Selector {
	var sopts selector.Options
	for _, opt := range opts {
		opt(&sopts)
	}
	if sopts.Registry == nil {
		sopts.Registry = registry.DefaultRegistry
	}
	return &dcSelector{sopts}
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
		client.Selector(DCSelector()),
	)

	fmt.Println("\n--- Call example ---\n")
	for i := 0; i < 10; i++ {
		call(i)
	}
}
