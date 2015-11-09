package etcd

import (
	etcd "github.com/coreos/etcd/client"
	"github.com/piemapping/go-micro/registry"
	"golang.org/x/net/context"
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
