package table

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
)

var (
	// ErrWatcherStopped is returned when routing table watcher has been stopped
	ErrWatcherStopped = errors.New("watcher stopped")
)

// EventType defines routing table event
type EventType int

const (
	// Insert is emitted when a new route has been inserted
	Insert EventType = iota
	// Delete is emitted when an existing route has been deleted
	Delete
	// Update is emitted when an existing route has been updated
	Update
)

// String returns string representation of the event
func (et EventType) String() string {
	switch et {
	case Insert:
		return "INSERT"
	case Delete:
		return "DELETE"
	case Update:
		return "UPDATE"
	default:
		return "UNKNOWN"
	}
}

// Event is returned by a call to Next on the watcher.
type Event struct {
	// Type defines type of event
	Type EventType
	// Timestamp is event timestamp
	Timestamp time.Time
	// Route is table route
	Route Route
}

// String prints human readable Event
func (e Event) String() string {
	return fmt.Sprintf("[EVENT] time: %s type: %s", e.Timestamp, e.Type)
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

type tableWatcher struct {
	opts    WatchOptions
	resChan chan *Event
	done    chan struct{}
}

// Next returns the next noticed action taken on table
// TODO: this needs to be thought through properly;
// right now we only allow to watch destination
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
	select {
	case <-w.done:
		return
	default:
		close(w.done)
	}
}

// String prints debug information
func (w tableWatcher) String() string {
	sb := &strings.Builder{}

	table := tablewriter.NewWriter(sb)
	table.SetHeader([]string{"Service"})

	data := []string{
		w.opts.Service,
	}
	table.Append(data)

	// render table into sb
	table.Render()

	return sb.String()
}
