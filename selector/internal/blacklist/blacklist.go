package blacklist

import (
	"math/rand"
	"sync"
	"time"

	"github.com/micro/go-micro/registry"
)

type blackListNode struct {
	age     time.Time
	id      string
	service string
	count   int
}

type BlackList struct {
	ttl  int
	exit chan bool

	sync.RWMutex
	bl map[string]blackListNode
}

var (
	// number of times we see an error before blacklisting
	count = 3

	// the ttl to blacklist for
	ttl = 30
)

func init() {
	rand.Seed(time.Now().Unix())
}

func (r *BlackList) purge() {
	now := time.Now()
	r.Lock()
	for k, v := range r.bl {
		if d := v.age.Sub(now); d.Seconds() < 0 {
			delete(r.bl, k)
		}
	}
	r.Unlock()
}

func (r *BlackList) run() {
	t := time.NewTicker(time.Duration(r.ttl) * time.Second)

	for {
		select {
		case <-r.exit:
			t.Stop()
			return
		case <-t.C:
			r.purge()
		}
	}
}

func (r *BlackList) Filter(services []*registry.Service) ([]*registry.Service, error) {
	var viableServices []*registry.Service

	r.RLock()

	for _, service := range services {
		var viableNodes []*registry.Node

		for _, node := range service.Nodes {
			n, ok := r.bl[node.Id]
			if !ok {
				// blacklist miss so add it
				viableNodes = append(viableNodes, node)
				continue
			}

			// got some blacklist info
			// skip the node if it exceeds count
			if n.count >= count {
				continue
			}

			// doesn't exceed count, still viable
			viableNodes = append(viableNodes, node)
		}

		if len(viableNodes) == 0 {
			continue
		}

		viableService := new(registry.Service)
		*viableService = *service
		viableService.Nodes = viableNodes
		viableServices = append(viableServices, viableService)
	}

	r.RUnlock()

	return viableServices, nil
}

func (r *BlackList) Mark(service string, node *registry.Node, err error) {
	r.Lock()
	defer r.Unlock()

	// reset when error is nil
	// basically closing the circuit
	if err == nil {
		delete(r.bl, node.Id)
		return
	}

	n, ok := r.bl[node.Id]
	if !ok {
		n = blackListNode{
			id:      node.Id,
			service: service,
		}
	}

	// mark it
	n.count++

	// set age to ttl seconds in future
	n.age = time.Now().Add(time.Duration(r.ttl) * time.Second)

	// save
	r.bl[node.Id] = n
}

func (r *BlackList) Reset(service string) {
	r.Lock()
	defer r.Unlock()

	for k, v := range r.bl {
		// delete every node that matches the service
		if v.service == service {
			delete(r.bl, k)
		}
	}
}

func (r *BlackList) Close() error {
	select {
	case <-r.exit:
		return nil
	default:
		close(r.exit)
	}
	return nil
}

func New() *BlackList {
	bl := &BlackList{
		ttl:  ttl,
		bl:   make(map[string]blackListNode),
		exit: make(chan bool),
	}

	go bl.run()
	return bl
}
