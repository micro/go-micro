package registry

import (
	"errors"
)

type memWatcher struct {
	wo   WatchOptions
	res  chan *Result
	exit chan bool
	id   string
}

func (m *memWatcher) Next() (*Result, error) {
	for {
		select {
		case r := <-m.res:
			if len(m.wo.Service) > 0 && m.wo.Service != r.Service.Name {
				continue
			}
			return r, nil
		case <-m.exit:
			return nil, errors.New("watcher stopped")
		}
	}
}

func (m *memWatcher) Stop() {
	select {
	case <-m.exit:
		return
	default:
		close(m.exit)
	}
}
