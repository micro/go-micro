package memory

import (
	"testing"

	"github.com/micro/go-micro/v3/registry"
)

func TestWatcher(t *testing.T) {
	w := &Watcher{
		id:   "test",
		res:  make(chan *registry.Result),
		exit: make(chan bool),
		wo: registry.WatchOptions{
			Domain: registry.WildcardDomain,
		},
	}

	go func() {
		w.res <- &registry.Result{
			Service: &registry.Service{Name: "foo"},
		}
	}()

	_, err := w.Next()
	if err != nil {
		t.Fatal("unexpected err", err)
	}

	w.Stop()

	if _, err := w.Next(); err == nil {
		t.Fatal("expected error on Next()")
	}
}
