package client

import (
	"math/rand"
	"time"

	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/registry"
)

// NodeSelector is used to retrieve a node to which a request
// should be routed. It takes a list of services and selects
// a single node. If a node cannot be selected it should return
// an error. A list of services is provided as a service may
// have 1 or more versions.
type NodeSelector func(service []*registry.Service) (*registry.Node, error)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Built in random hashed node selector
func nodeSelector(service []*registry.Service) (*registry.Node, error) {
	if len(service) == 0 {
		return nil, errors.NotFound("go.micro.client", "Service not found")
	}

	i := rand.Int()
	j := i % len(service)

	if len(service[j].Nodes) == 0 {
		return nil, errors.NotFound("go.micro.client", "Service not found")
	}

	n := i % len(service[j].Nodes)
	return service[j].Nodes[n], nil
}
