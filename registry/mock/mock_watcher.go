package mock

import (
	"errors"

	"github.com/micro/go-micro/registry"
)

type mockWatcher struct {
	exit chan bool
	opts registry.WatchOptions
}

func (m *mockWatcher) Next() (*registry.Result, error) {
	// not implement so we just block until exit
	select {
	case <-m.exit:
		return nil, errors.New("watcher stopped")
	}
}

func (m *mockWatcher) Stop() {
	select {
	case <-m.exit:
		return
	default:
		close(m.exit)
	}
}
