package router

import (
	"errors"
)

var (
	// ErrWatcherStopped is returned when routing table watcher has been stopped
	ErrWatcherStopped = errors.New("routing table watcher stopped")
)

// WatchOption is used to define what routes to watch in the table
type WatchOption func(*WatchOptions)

// Watcher defines routing table watcher interface
// Watcher returns updates to the routing table
type Watcher interface {
	// Next is a blocking call that returns watch result
	Next() (*Result, error)
	// Stop stops watcher
	Stop()
}

// Result is returned by a call to Next on the watcher.
type Result struct {
	// Action is routing table action which is either of add, remove or update
	Action string
	// Route is table rout
	Route Route
}

// WatchOptions are table watcher options
type WatchOptions struct {
	// Specify destination address to watch
	DestAddr string
	// Specify network to watch
	Network string
}

// WatchDestAddr sets what destination to watch
// Destination is usually microservice name
func WatchDestAddr(a string) WatchOption {
	return func(o *WatchOptions) {
		o.DestAddr = a
	}
}

// WatchNetwork sets what network to watch
func WatchNetwork(n string) WatchOption {
	return func(o *WatchOptions) {
		o.Network = n
	}
}

type tableWatcher struct {
	opts    WatchOptions
	resChan chan *Result
	done    chan struct{}
}

// TODO: this needs to be thought through properly
// Next returns the next noticed action taken on table
func (w *tableWatcher) Next() (*Result, error) {
	for {
		select {
		case res := <-w.resChan:
			switch w.opts.DestAddr {
			case "*", "":
				if w.opts.Network == "*" || w.opts.Network == res.Route.Options().Network {
					return res, nil
				}
			case res.Route.Options().DestAddr:
				if w.opts.Network == "*" || w.opts.Network == res.Route.Options().Network {
					return res, nil
				}
			}
			// ignore if no match is found
			continue
		case <-w.done:
			return nil, ErrWatcherStopped
		}
	}
}

// Stop stops routing table watcher
func (w *tableWatcher) Stop() {
	select {
	case <-w.done:
		return
	default:
		close(w.done)
	}
}
