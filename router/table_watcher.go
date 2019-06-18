package router

import (
	"errors"
)

var (
	// ErrWatcherStopped is returned when routing table watcher has been stopped
	ErrWatcherStopped = errors.New("routing table watcher stopped")
)

// EventType defines routing table event
type EventType int

const (
	// CreateEvent is emitted when new route has been created
	CreateEvent EventType = iota
	// DeleteEvent is emitted when an existing route has been deleted
	DeleteEvent
	// UpdateEvent is emitted when a routing table has been updated
	UpdateEvent
)

// String returns string representation of the event
func (et EventType) String() string {
	switch et {
	case CreateEvent:
		return "CREATE"
	case DeleteEvent:
		return "DELETE"
	case UpdateEvent:
		return "UPDATE"
	default:
		return "UNKNOWN"
	}
}

// Event is returned by a call to Next on the watcher.
type Event struct {
	// Type defines type of event
	Type EventType
	// Route is table rout
	Route Route
}

// WatchOption is used to define what routes to watch in the table
type WatchOption func(*WatchOptions)

// Watcher defines routing table watcher interface
// Watcher returns updates to the routing table
type Watcher interface {
	// Next is a blocking call that returns watch result
	Next() (*Event, error)
	// Stop stops watcher
	Stop()
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
	resChan chan *Event
	done    chan struct{}
}

// Next returns the next noticed action taken on table
// TODO: this needs to be thought through properly
// we are aiming to provide the same watch options Query() provides
func (w *tableWatcher) Next() (*Event, error) {
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
