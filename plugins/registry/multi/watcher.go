package multi

import (
	"sync"

	"github.com/micro/go-micro/v2/registry"
)

type multiWatcher struct {
	wo   registry.WatchOptions
	w    []registry.Watcher
	next chan *registry.Result
	stop chan bool
}

func newMultiWatcher(r []registry.Registry, opts ...registry.WatchOption) (registry.Watcher, error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	mw := &multiWatcher{
		wo:   wo,
		next: make(chan *registry.Result),
		stop: make(chan bool),
	}

	for _, wr := range r {
		w, err := wr.Watch(opts...)
		if err != nil {
			return nil, err
		}
		mw.w = append(mw.w, w)
	}

	return mw, nil
}

func (mw *multiWatcher) Next() (*registry.Result, error) {
	cerr := make(chan error)

	for _, wt := range mw.w {
		go func(w registry.Watcher) {
			r, err := w.Next()
			if err != nil && err != registry.ErrNotFound {
				cerr <- err
			}
			mw.next <- r
		}(wt)
	}

	for {
		select {
		case err := <-cerr:
			return nil, err
		case r, ok := <-mw.next:
			if !ok {
				return nil, registry.ErrWatcherStopped
			}
			nr := &registry.Result{}
			*nr = *r
			return nr, nil
		case <-mw.stop:
			return nil, registry.ErrWatcherStopped
		}
	}
}

func (mw *multiWatcher) Stop() {
	var wg sync.WaitGroup
	wg.Add(len(mw.w))

	for _, w := range mw.w {
		go func(w registry.Watcher) {
			w.Stop()
			wg.Done()
		}(w)
	}

	wg.Wait()
	select {
	case <-mw.stop:
		return
	default:
		close(mw.stop)
	}
}
