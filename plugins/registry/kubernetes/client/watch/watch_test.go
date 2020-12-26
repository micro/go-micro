package watch

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var actions = []string{
	`{"type": "create", "object":{"foo": "bar"}}`,
	`{"type": "delete", INVALID}`,
	`{"type": "update", "object":{"foo": {"foo": "bar"}}}`,
	`{"type": "delete", "object":null}`,
}

func TestBodyWatcher(t *testing.T) {
	// set up server with handler to flush strings from ch.
	ch := make(chan string)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected ResponseWriter to be a flusher")
		}

		fmt.Fprintf(w, "\n")
		flusher.Flush()

		for v := range ch {
			fmt.Fprintf(w, "%s\n", v)
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Fatalf("did not expect NewRequest to return err: %v", err)
	}

	// setup body watcher
	w, err := NewBodyWatcher(req, http.DefaultClient)
	if err != nil {
		t.Fatalf("did not expect NewBodyWatcher to return %v", err)
	}

	<-time.After(time.Second)

	// send action strings in, and expect result back
	ch <- actions[0]
	if r := <-w.ResultChan(); r.Type != "create" {
		t.Fatalf("expected result to be create")
	}

	ch <- actions[1] // should be ignored as its invalid json
	ch <- actions[2]
	if r := <-w.ResultChan(); r.Type != "update" {
		t.Fatalf("expected result to be update")
	}

	ch <- actions[3]
	if r := <-w.ResultChan(); r.Type != "delete" {
		t.Fatalf("expected result to be delete")
	}

	// stop should clean up all channels.
	w.Stop()
	close(ch)
}
