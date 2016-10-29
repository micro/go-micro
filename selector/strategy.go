package selector

import (
	"math/rand"
	"sync"
	"time"

	"github.com/micro/go-micro/registry"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Random is a random strategy algorithm for node selection
func Random(services []*registry.Service) Next {
	var nodes []*registry.Node

	for _, service := range services {
		nodes = append(nodes, service.Nodes...)
	}

	return func() (*registry.Node, error) {
		if count := len(nodes); count == 0 {
			return nil, ErrNoneAvailable
		} else {
			i := rand.Int() % count
			return nodes[i], nil
		}
	}
}

// RoundRobin is a roundrobin strategy algorithm for node selection
func RoundRobin(services []*registry.Service) Next {
	var nodes []*registry.Node

	for _, service := range services {
		nodes = append(nodes, service.Nodes...)
	}

	// Shuffle: No need to start from the first
	for i := range nodes {
		j := rand.Intn(i + 1)
		nodes[i], nodes[j] = nodes[j], nodes[i]
	}

	var i int
	var mtx sync.Mutex

	return func() (*registry.Node, error) {
		if count := len(nodes); count == 0 {
			return nil, ErrNoneAvailable
		} else {
			mtx.Lock()
			node := nodes[i%count]
			i++
			mtx.Unlock()

			return node, nil
		}
	}
}
