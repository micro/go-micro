package memory

import (
	"testing"

	"github.com/micro/go-micro/registry"
)

func TestWatcher(t *testing.T) {
	w := &Watcher{
		id:   "test",
		res:  make(chan *registry.Event),
		exit: make(chan bool),
	}

	go func() {
		w.res <- &registry.Event{}
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
