package router

import (
	"errors"
	"strings"

	"github.com/olekukonko/tablewriter"
)

var (
	// ErrWatcherStopped is returned when routing table watcher has been stopped
	ErrWatcherStopped = errors.New("watcher stopped")
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
	// Chan returns event channel
	Chan() (<-chan *Event, error)
	// Stop stops watcher
	Stop()
}

// WatchOptions are table watcher options
type WatchOptions struct {
	// Specify destination address to watch
	Destination string
}

// WatchDestination sets what destination to watch
// Destination is usually microservice name
func WatchDestination(d string) WatchOption {
	return func(o *WatchOptions) {
		o.Destination = d
	}
}

type tableWatcher struct {
	opts    WatchOptions
	resChan chan *Event
	done    chan struct{}
}

// Next returns the next noticed action taken on table
// TODO: this needs to be thought through properly; we only allow watching particular route destination for now
func (w *tableWatcher) Next() (*Event, error) {
	for {
		select {
		case res := <-w.resChan:
			switch w.opts.Destination {
			case "*", "":
				return res, nil
			default:
				if w.opts.Destination == res.Route.Destination {
					return res, nil
				}
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
	select {
	case <-w.done:
		return
	default:
		close(w.done)
	}
}

// String prints debug information
func (w *tableWatcher) String() string {
	sb := &strings.Builder{}

	table := tablewriter.NewWriter(sb)
	table.SetHeader([]string{"Destination"})

	data := []string{
		w.opts.Destination,
	}
	table.Append(data)

	// render table into sb
	table.Render()

	return sb.String()
}
