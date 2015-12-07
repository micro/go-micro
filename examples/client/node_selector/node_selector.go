package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/cmd"
	"github.com/micro/go-micro/errors"
	example "github.com/micro/go-micro/examples/server/proto/example"
	"github.com/micro/go-micro/registry"
	"golang.org/x/net/context"
)

func init() {
	rand.Seed(time.Now().Unix())
}

// Built in random hashed node selector
type nodeSelector struct {
	r registry.Registry
}

func (n *nodeSelector) Retrieve(ctx context.Context, req client.Request) (*registry.Node, error) {
	service, err := n.r.GetService(req.Service())
	if err != nil {
		return nil, errors.InternalServerError("go.micro.client", err.Error())
	}

	if len(service) == 0 {
		return nil, errors.NotFound("go.micro.client", "Service not found")
	}

	i := rand.Int()
	j := i % len(service)

	if len(service[j].Nodes) == 0 {
		return nil, errors.NotFound("go.micro.client", "Service not found")
	}

	k := i % len(service[j].Nodes)
	return service[j].Nodes[k], nil
}

func (n *nodeSelector) Response(node *registry.Node, err error) {
	return
}

func (n *nodeSelector) Reset() {
	return
}

// Return a new random node selector
func RandomSelector(r registry.Registry) client.NodeSelector {
	return &nodeSelector{r}
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
		client.Select(RandomSelector),
	)

	fmt.Println("\n--- Call example ---\n")
	for i := 0; i < 10; i++ {
		call(i)
	}
}
