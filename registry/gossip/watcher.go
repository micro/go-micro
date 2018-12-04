package gossip

import (
	"github.com/micro/go-micro/registry"
)

type watcher struct {
	id   string
	srv  string
	ch   chan *registry.Result
	exit chan bool
	fn   func()
}

func (w *watcher) Next() (*registry.Result, error) {
	for {
		select {
		case r := <-w.ch:
			if r.Service == nil {
				continue
			}
			if len(w.srv) > 0 && (r.Service.Name != w.srv) {
				continue
			}
			return r, nil
		case <-w.exit:
			return nil, registry.ErrWatcherStopped
		}
	}
}

func (w *watcher) Stop() {
	select {
	case <-w.exit:
		return
	default:
		close(w.exit)
		w.fn()
	}
}
