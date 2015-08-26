package etcd

import (
	"github.com/coreos/go-etcd/etcd"
	"github.com/kynrai/go-micro/registry"
)

type etcdWatcher struct {
	registry *etcdRegistry
	stop     chan bool
}

func newEtcdWatcher(r *etcdRegistry) (registry.Watcher, error) {
	ew := &etcdWatcher{
		registry: r,
		stop:     make(chan bool),
	}

	ch := make(chan *etcd.Response)

	go r.client.Watch(prefix, 0, true, ch, ew.stop)
	go ew.watch(ch)

	return ew, nil
}

func (e *etcdWatcher) watch(ch chan *etcd.Response) {
	for rsp := range ch {
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
				e.registry.services[s.Name] = s
			}
			e.registry.Unlock()
			continue
		}

		switch rsp.Action {
		case "delete":
			var nodes []*registry.Node
			for _, node := range service.Nodes {
				var seen bool
				for _, n := range s.Nodes {
					if node.Id == n.Id {
						seen = true
						break
					}
				}
				if !seen {
					nodes = append(nodes, node)
				}
			}
			service.Nodes = nodes
		case "create":
			service.Nodes = append(service.Nodes, s.Nodes...)
		}

		e.registry.Unlock()
	}
}

func (ew *etcdWatcher) Stop() {
	ew.stop <- true
}
