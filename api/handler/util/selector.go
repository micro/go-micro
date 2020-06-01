package util

import (
	"math/rand"

	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/selector"
)

// Selector is a hack for selection
func Selector(nodes []*registry.Node) selector.Selector {
	return &apiSelector{nodes}
}

type apiSelector struct {
	nodes []*registry.Service
}

func (s *apiSelector) Select(...selector.Node) (selector.Node, error) {
	if len(s.nodes) == 0 {
		return nil, selector.ErrNoneAvailable
	}
	if len(s.nodes) == 1 {
		return s.nodes[0], nil
	}
	return s.nodes[rand.Intn(len(s.nodes)-1)], nil
}

func (s *apiSelector) Record(selector.Node, selector.Result) error {
	return nil
}

func (s *apiSelector) String() string {
	return "api"
}
