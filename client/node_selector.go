package client

import (
	"math/rand"
	"time"

	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/registry"
	"golang.org/x/net/context"
)

// NodeSelector is used to Retrieve a node to which a request
// should be routed. It takes context and Request and returns a
// single node. If a node cannot be selected it should return
// an error. Response is called to inform the selector of the
// response from a client call. Reset is called to zero out
// any state.
type NodeSelector interface {
	Retrieve(context.Context, Request) (*registry.Node, error)
	Response(*registry.Node, error)
	Reset()
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Built in random hashed node selector
type nodeSelector struct {
	r registry.Registry
}

func (n *nodeSelector) Retrieve(ctx context.Context, req Request) (*registry.Node, error) {
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
