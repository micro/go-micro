package registry

import (
	"sync"

	"github.com/micro/go-micro/v3/router"
)

// tableWatcher implements routing table Watcher
type tableWatcher struct {
	sync.RWMutex
	id      string
	opts    router.WatchOptions
	resChan chan *router.Event
	done    chan struct{}
}

// Next returns the next noticed action taken on table
// TODO: right now we only allow to watch particular service
func (w *tableWatcher) Next() (*router.Event, error) {
	for {
		select {
		case res := <-w.resChan:
			switch w.opts.Service {
			case res.Route.Service, "*":
				return res, nil
			default:
				continue
			}
		case <-w.done:
			return nil, router.ErrWatcherStopped
		}
	}
}

// Chan returns watcher events channel
func (w *tableWatcher) Chan() (<-chan *router.Event, error) {
	return w.resChan, nil
}

// Stop stops routing table watcher
func (w *tableWatcher) Stop() {
	w.Lock()
	defer w.Unlock()

	select {
	case <-w.done:
		return
	default:
		close(w.done)
	}
}
