package client

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/micro/go-micro/v2/util/kubernetes/api"
)

const (
	// EventTypes used
	Added    EventType = "ADDED"
	Modified EventType = "MODIFIED"
	Deleted  EventType = "DELETED"
	Error    EventType = "ERROR"
)

// Watcher is used to watch for events
type Watcher interface {
	// A channel of events
	Chan() <-chan Event
	// Stop the watcher
	Stop()
}

// EventType defines the possible types of events.
type EventType string

// Event represents a single event to a watched resource.
type Event struct {
	Type   EventType       `json:"type"`
	Object json.RawMessage `json:"object"`
}

// bodyWatcher scans the body of a request for chunks
type bodyWatcher struct {
	results chan Event
	cancel  func()
	stop    chan bool
	res     *http.Response
	req     *api.Request
}

// Changes returns the results channel
func (wr *bodyWatcher) Chan() <-chan Event {
	return wr.results
}

// Stop cancels the request
func (wr *bodyWatcher) Stop() {
	select {
	case <-wr.stop:
		return
	default:
		// cancel the request
		wr.cancel()
		// stop the watcher
		close(wr.stop)
	}
}

func (wr *bodyWatcher) stream() {
	reader := bufio.NewReader(wr.res.Body)

	go func() {
		for {
			// read a line
			b, err := reader.ReadBytes('\n')
			if err != nil {
				return
			}

			// send the event
			var event Event
			if err := json.Unmarshal(b, &event); err != nil {
				continue
			}

			select {
			case <-wr.stop:
				return
			case wr.results <- event:
			}
		}
	}()
}

// newWatcher creates a k8s body watcher for
// a given http request
func newWatcher(req *api.Request) (Watcher, error) {
	// set request context so we can cancel the request
	ctx, cancel := context.WithCancel(context.Background())
	req.Context(ctx)

	// do the raw request
	res, err := req.Raw()
	if err != nil {
		cancel()
		return nil, err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		cancel()
		// close the response body
		res.Body.Close()
		// return an error
		return nil, errors.New(res.Request.URL.String() + ": " + res.Status)
	}

	wr := &bodyWatcher{
		results: make(chan Event),
		stop:    make(chan bool),
		cancel:  cancel,
		req:     req,
		res:     res,
	}

	go wr.stream()

	return wr, nil
}
