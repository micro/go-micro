package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/cmd"
	c "github.com/micro/go-micro/context"
	"github.com/micro/go-micro/errors"
	example "github.com/micro/go-micro/examples/server/proto/example"
	"github.com/micro/go-micro/registry"
	"golang.org/x/net/context"
)

func init() {
	rand.Seed(time.Now().Unix())
}

// A random node selector
func randomSelector(s []*registry.Service) (*registry.Node, error) {
	if len(s) == 0 {
		return nil, errors.NotFound("go.micro.client", "Service not found")
	}

	i := rand.Int()
	j := i % len(s)

	if len(s[j].Nodes) == 0 {
		return nil, errors.NotFound("go.micro.client", "Service not found")
	}

	n := i % len(s[j].Nodes)
	return s[j].Nodes[n], nil
}

// Wraps the node selector so that it will log what node was selected
func wrapSelector(fn client.NodeSelector) client.NodeSelector {
	return func(s []*registry.Service) (*registry.Node, error) {
		n, err := fn(s)
		if err != nil {
			return nil, err
		}
		fmt.Printf("Selected node %v\n", n)
		return n, nil
	}
}

func call(i int) {
	// Create new request to service go.micro.srv.example, method Example.Call
	req := client.NewRequest("go.micro.srv.example", "Example.Call", &example.Request{
		Name: "John",
	})

	// create context with metadata
	ctx := c.WithMetadata(context.Background(), map[string]string{
		"X-User-Id": "john",
		"X-From-Id": "script",
	})

	rsp := &example.Response{}

	// Call service
	if err := client.Call(ctx, req, rsp); err != nil {
		fmt.Println("call err: ", err, rsp)
		return
	}

	fmt.Println("Call:", i, "rsp:", rsp.Msg)
}

func main() {
	cmd.Init()

	client.DefaultClient = client.NewClient(
		client.Selector(wrapSelector(randomSelector)),
	)

	fmt.Println("\n--- Call example ---\n")
	for i := 0; i < 10; i++ {
		call(i)
	}
}
