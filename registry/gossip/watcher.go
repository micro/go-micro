package gossip

import (
	"github.com/micro/go-micro/registry"
)

type gossipWatcher struct {
	wo   registry.WatchOptions
	next chan *registry.Event
	stop chan bool
}

func newGossipWatcher(ch chan *registry.Event, stop chan bool, opts ...registry.WatchOption) (registry.Watcher, error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	return &gossipWatcher{
		wo:   wo,
		next: ch,
		stop: stop,
	}, nil
}

func (g *gossipWatcher) Chan() (<-chan *registry.Event, error) {
	ch := make(chan *registry.Event, 32)

	// spinup a watcher
	go func() {
		for {
			ev, err := g.Next()
			if err != nil {
				close(ch)
				return
			}
			ch <- ev
		}
	}()

	return ch, nil
}

func (m *gossipWatcher) Next() (*registry.Event, error) {
	for {
		select {
		case r, ok := <-m.next:
			if !ok {
				return nil, registry.ErrWatcherStopped
			}
			// check watch options
			if len(m.wo.Service) > 0 && r.Service.Name != m.wo.Service {
				continue
			}
			nr := &registry.Event{}
			*nr = *r
			return nr, nil
		case <-m.stop:
			return nil, registry.ErrWatcherStopped
		}
	}
}

func (m *gossipWatcher) Stop() {
	select {
	case <-m.stop:
		return
	default:
		close(m.stop)
	}
}
