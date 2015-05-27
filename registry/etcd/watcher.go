package etcd

import (
	"github.com/coreos/go-etcd/etcd"
	"github.com/myodc/go-micro/registry"
)

type etcdWatcher struct {
	registry *etcdRegistry
	stop     chan bool
}

func newEtcdWatcher(r *etcdRegistry) *etcdWatcher {
	ew := &etcdWatcher{
		registry: r,
		stop:     make(chan bool),
	}

	ch := make(chan *etcd.Response)

	go r.client.Watch(prefix, 0, true, ch, ew.stop)

	go func() {
		for rsp := range ch {
			if rsp.Node.Dir {
				continue
			}

			s := decode(rsp.Node.Value)
			if s == nil {
				continue
			}

			r.Lock()

			service, ok := r.services[s.Name]
			if !ok {
				if rsp.Action == "create" {
					r.services[s.Name] = s
				}
				r.Unlock()
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
			r.Unlock()
		}
	}()

	return ew
}

func (ew *etcdWatcher) Stop() {
	ew.stop <- true
}
