package etcd

import (
	etcd "github.com/coreos/etcd/client"
	"github.com/micro/go-micro/registry"
	"golang.org/x/net/context"
)

type etcdWatcher struct {
	registry *etcdRegistry
	stop     chan bool
}

func addNodes(old, neu []*registry.Node) []*registry.Node {
	for _, n := range neu {
		var seen bool
		for i, o := range old {
			if o.Id == n.Id {
				seen = true
				old[i] = n
				break
			}
		}
		if !seen {
			old = append(old, n)
		}
	}
	return old
}

func addServices(old, neu []*registry.Service) []*registry.Service {
	for _, s := range neu {
		var seen bool
		for i, o := range old {
			if o.Version == s.Version {
				s.Nodes = addNodes(o.Nodes, s.Nodes)
				seen = true
				old[i] = s
				break
			}
		}
		if !seen {
			old = append(old, s)
		}
	}
	return old
}

func delNodes(old, del []*registry.Node) []*registry.Node {
	var nodes []*registry.Node
	for _, o := range old {
		var rem bool
		for _, n := range del {
			if o.Id == n.Id {
				rem = true
				break
			}
		}
		if !rem {
			nodes = append(nodes, o)
		}
	}
	return nodes
}

func delServices(old, del []*registry.Service) []*registry.Service {
	var services []*registry.Service
	for i, o := range old {
		var rem bool
		for _, s := range del {
			if o.Version == s.Version {
				old[i].Nodes = delNodes(o.Nodes, s.Nodes)
				if len(old[i].Nodes) == 0 {
					rem = true
				}
			}
		}
		if !rem {
			services = append(services, o)
		}
	}
	return services
}

func newEtcdWatcher(r *etcdRegistry) (registry.Watcher, error) {
	ew := &etcdWatcher{
		registry: r,
		stop:     make(chan bool),
	}

	w := r.client.Watcher(prefix, &etcd.WatcherOptions{AfterIndex: 0, Recursive: true})

	c := context.Background()
	ctx, cancel := context.WithCancel(c)

	go func() {
		<-ew.stop
		cancel()
	}()

	go ew.watch(ctx, w)

	return ew, nil
}

func (e *etcdWatcher) watch(ctx context.Context, w etcd.Watcher) {
	for {
		rsp, err := w.Next(ctx)
		if err != nil && ctx.Err() != nil {
			return
		}

		if rsp.Node.Dir {
			continue
		}

		s := decode(rsp.Node.Value)
		if s == nil {
			continue
		}

		e.registry.Lock()

		service, ok := e.registry.services[s.Name]
		if !ok {
			if rsp.Action == "create" {
				e.registry.services[s.Name] = []*registry.Service{s}
			}
			e.registry.Unlock()
			continue
		}

		switch rsp.Action {
		case "delete":
			services := delServices(service, []*registry.Service{s})
			if len(services) > 0 {
				e.registry.services[s.Name] = services
			} else {
				delete(e.registry.services, s.Name)
			}
		case "create":
			e.registry.services[s.Name] = addServices(service, []*registry.Service{s})
		}

		e.registry.Unlock()
	}
}

func (ew *etcdWatcher) Stop() {
	ew.stop <- true
}
