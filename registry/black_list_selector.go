package registry

import (
	"math/rand"
	"sync"
	"time"
)

type blackListNode struct {
	age     time.Time
	id      string
	service string
}

type blackListSelector struct {
	so   SelectorOptions
	ttl  int64
	exit chan bool
	once sync.Once

	sync.RWMutex
	bl map[string]blackListNode
}

func init() {
	rand.Seed(time.Now().Unix())
}

func (r *blackListSelector) purge() {
	now := time.Now()
	r.Lock()
	for k, v := range r.bl {
		if d := v.age.Sub(now); d.Seconds() < 0 {
			delete(r.bl, k)
		}
	}
	r.Unlock()
}

func (r *blackListSelector) run() {
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

func (r *blackListSelector) Select(service string, opts ...SelectOption) (SelectNext, error) {
	var sopts SelectOptions
	for _, opt := range opts {
		opt(&sopts)
	}

	// get the service
	services, err := r.so.Registry.GetService(service)
	if err != nil {
		return nil, err
	}

	// apply the filters
	for _, filter := range sopts.Filters {
		services = filter(services)
	}

	// if there's nothing left, return
	if len(services) == 0 {
		return nil, ErrNotFound
	}

	var nodes []*Node

	for _, service := range services {
		for _, node := range service.Nodes {
			nodes = append(nodes, node)
		}
	}

	if len(nodes) == 0 {
		return nil, ErrNotFound
	}

	return func() (*Node, error) {
		var viable []*Node

		r.RLock()
		for _, node := range nodes {
			if _, ok := r.bl[node.Id]; !ok {
				viable = append(viable, node)
			}
		}
		r.RUnlock()

		if len(viable) == 0 {
			return nil, ErrNoneAvailable
		}

		return viable[rand.Int()%len(viable)], nil
	}, nil
}

func (r *blackListSelector) Mark(service string, node *Node, err error) {
	r.Lock()
	defer r.Unlock()
	if err == nil {
		delete(r.bl, node.Id)
		return
	}

	r.bl[node.Id] = blackListNode{
		age:     time.Now().Add(time.Duration(r.ttl) * time.Second),
		id:      node.Id,
		service: service,
	}
	return
}

func (r *blackListSelector) Reset(service string) {
	r.Lock()
	defer r.Unlock()
	for k, v := range r.bl {
		if v.service == service {
			delete(r.bl, k)
		}
	}
	return
}

func (r *blackListSelector) Close() error {
	r.once.Do(func() {
		close(r.exit)
	})
	return nil
}

func NewBlackListSelector(opts ...SelectorOption) Selector {
	var sopts SelectorOptions

	for _, opt := range opts {
		opt(&sopts)
	}

	if sopts.Registry == nil {
		sopts.Registry = DefaultRegistry
	}

	var once sync.Once
	bl := &blackListSelector{
		once: once,
		so:   sopts,
		ttl:  60,
		bl:   make(map[string]blackListNode),
		exit: make(chan bool),
	}

	go bl.run()

	return bl
}
