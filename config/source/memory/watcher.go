package memory

import (
	"go-micro.dev/v5/config/source"
)

type watcher struct {
	Updates chan *source.ChangeSet
	Source  *memory
	Id      string
}

func (w *watcher) Next() (*source.ChangeSet, error) {
	cs := <-w.Updates
	return cs, nil
}

func (w *watcher) Stop() error {
	w.Source.Lock()
	delete(w.Source.Watchers, w.Id)
	w.Source.Unlock()
	return nil
}
