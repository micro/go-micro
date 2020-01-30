package service

import (
	"io"
	"sync"
	"time"

	"github.com/micro/go-micro/v2/router"
	pb "github.com/micro/go-micro/v2/router/service/proto"
)

type watcher struct {
	sync.RWMutex
	opts    router.WatchOptions
	resChan chan *router.Event
	done    chan struct{}
	stream  pb.Router_WatchService
}

func newWatcher(rsp pb.Router_WatchService, opts router.WatchOptions) (*watcher, error) {
	w := &watcher{
		opts:    opts,
		resChan: make(chan *router.Event),
		done:    make(chan struct{}),
		stream:  rsp,
	}

	go func() {
		for {
			select {
			case <-w.done:
				return
			default:
				if err := w.watch(rsp); err != nil {
					w.Stop()
					return
				}
			}
		}
	}()

	return w, nil
}

// watchRouter watches router and send events to all registered watchers
func (w *watcher) watch(stream pb.Router_WatchService) error {
	var watchErr error

	for {
		resp, err := stream.Recv()
		if err != nil {
			if err != io.EOF {
				watchErr = err
			}
			break
		}

		route := router.Route{
			Service: resp.Route.Service,
			Address: resp.Route.Address,
			Gateway: resp.Route.Gateway,
			Network: resp.Route.Network,
			Link:    resp.Route.Link,
			Metric:  resp.Route.Metric,
		}

		event := &router.Event{
			Id:        resp.Id,
			Type:      router.EventType(resp.Type),
			Timestamp: time.Unix(0, resp.Timestamp),
			Route:     route,
		}

		select {
		case w.resChan <- event:
		case <-w.done:
		}
	}

	return watchErr
}

// Next is a blocking call that returns watch result
func (w *watcher) Next() (*router.Event, error) {
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
func (w *watcher) Chan() (<-chan *router.Event, error) {
	return w.resChan, nil
}

// Stop stops watcher
func (w *watcher) Stop() {
	w.Lock()
	defer w.Unlock()

	select {
	case <-w.done:
		return
	default:
		w.stream.Close()
		close(w.done)
	}
}
