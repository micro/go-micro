package router

import (
	"errors"
	"sync"
	"time"
)

var (
	// ErrWatcherStopped is returned when routing table watcher has been stopped
	ErrWatcherStopped = errors.New("watcher stopped")
)

// EventType defines routing table event
type EventType int

const (
	// Create is emitted when a new route has been created
	Create EventType = iota
	// Delete is emitted when an existing route has been deleted
	Delete
	// Update is emitted when an existing route has been updated
	Update
)

// String returns human readable event type
func (t EventType) String() string {
	switch t {
	case Create:
		return "create"
	case Delete:
		return "delete"
	case Update:
		return "update"
	default:
		return "unknown"
	}
}

// Event is returned by a call to Next on the watcher.
type Event struct {
	// Unique id of the event
	Id string
	// Type defines type of event
	Type EventType
	// Timestamp is event timestamp
	Timestamp time.Time
	// Route is table route
	Route Route
}

// Watcher defines routing table watcher interface
// Watcher returns updates to the routing table
type Watcher interface {
	// Next is a blocking call that returns watch result
	Next() (*Event, error)
	// Chan returns event channel
	Chan() (<-chan *Event, error)
	// Stop stops watcher
	Stop()
}

// WatchOption is used to define what routes to watch in the table
type WatchOption func(*WatchOptions)

// WatchOptions are table watcher options
// TODO: expand the options to watch based on other criteria
type WatchOptions struct {
	// Service allows to watch specific service routes
	Service string
}

// WatchService sets what service routes to watch
// Service is the microservice name
func WatchService(s string) WatchOption {
	return func(o *WatchOptions) {
		o.Service = s
	}
}

// tableWatcher implements routing table Watcher
type tableWatcher struct {
	sync.RWMutex
	id      string
	opts    WatchOptions
	resChan chan *Event
	done    chan struct{}
}

// Next returns the next noticed action taken on table
// TODO: right now we only allow to watch particular service
func (w *tableWatcher) Next() (*Event, error) {
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
			return nil, ErrWatcherStopped
		}
	}
}

// Chan returns watcher events channel
func (w *tableWatcher) Chan() (<-chan *Event, error) {
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
