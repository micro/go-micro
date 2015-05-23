package kubernetes

import (
	"github.com/myodc/go-micro/registry"
)

type service struct {
	name  string
	nodes []*node
}

func (s *service) Name() string {
	return s.name
}

func (s *service) Nodes() []registry.Node {
	var nodes []registry.Node

	for _, node := range s.nodes {
		nodes = append(nodes, node)
	}

	return nodes
}
