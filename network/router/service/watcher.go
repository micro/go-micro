package service

import (
	"sync"

	"github.com/micro/go-micro/network/router"
)

type svcWatcher struct {
	sync.RWMutex
	opts    router.WatchOptions
	resChan chan *router.Event
	done    chan struct{}
}

// Next is a blocking call that returns watch result
func (w *svcWatcher) Next() (*router.Event, error) {
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

// Chan returns event channel
func (w *svcWatcher) Chan() (<-chan *router.Event, error) {
	return w.resChan, nil
}

// Stop stops watcher
func (w *svcWatcher) Stop() {
	w.Lock()
	defer w.Unlock()

	select {
	case <-w.done:
		return
	default:
		close(w.done)
	}
}
