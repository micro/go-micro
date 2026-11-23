package watch

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
)

// bodyWatcher scans the body of a request for chunks.
type bodyWatcher struct {
	ctx     context.Context
	stop    context.CancelFunc
	results chan Event
	res     *http.Response
	req     *http.Request
}

// Changes returns the results channel.
func (wr *bodyWatcher) ResultChan() <-chan Event {
	return wr.results
}

// Stop cancels the request.
func (wr *bodyWatcher) Stop() {
	select {
	case <-wr.ctx.Done():
		return
	default:
		wr.stop()
	}
}

func (wr *bodyWatcher) stream() {
	reader := bufio.NewReader(wr.res.Body)

	// ignore first few messages from stream,
	// as they are usually old.
	var ignore atomic.Bool

	go func() {
		<-time.After(time.Second)
		ignore.Store(false)
	}()

	go func() {
		//nolint:errcheck
		defer wr.res.Body.Close()
	out:
		for {
			// Read a line
			b, err := reader.ReadBytes('\n')
			if err != nil {
				break
			}

			// Ignore for the first second
			if ignore.Load() {
				continue
			}

			// Send the event
			var event Event
			if err := json.Unmarshal(b, &event); err != nil {
				continue
			}

			select {
			case <-wr.ctx.Done():
				break out
			case wr.results <- event:
			}
		}

		close(wr.results)
		// stop the watcher
		wr.Stop()
	}()
}

// NewBodyWatcher creates a k8s body watcher for a given http request.
func NewBodyWatcher(req *http.Request, client *http.Client) (Watch, error) {
	ctx, cancel := context.WithCancel(context.Background())

	req = req.WithContext(ctx)

	//nolint:bodyclose
	res, err := client.Do(req)
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "body watcher failed to make http request")
	}

	wr := &bodyWatcher{
		ctx:     ctx,
		results: make(chan Event),
		stop:    cancel,
		req:     req,
		res:     res,
	}

	go wr.stream()

	return wr, nil
}
