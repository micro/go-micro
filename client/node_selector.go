package client

import (
	"math/rand"
	"time"

	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/registry"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

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
