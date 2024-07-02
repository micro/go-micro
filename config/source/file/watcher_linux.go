//go:build linux
// +build linux

package file

import (
	"os"

	"github.com/fsnotify/fsnotify"
	"go-micro.dev/v5/config/source"
)

type watcher struct {
	f *file

	fw *fsnotify.Watcher
}

func newWatcher(f *file) (source.Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	fw.Add(f.path)

	return &watcher{
		f:  f,
		fw: fw,
	}, nil
}

func (w *watcher) Next() (*source.ChangeSet, error) {
	// try get the event
	select {
	case event, ok := <-w.fw.Events:
		// check if channel was closed (i.e. Watcher.Close() was called).
		if !ok {
			return nil, source.ErrWatcherStopped
		}

		if event.Has(fsnotify.Rename) {
			// check existence of file, and add watch again
			_, err := os.Stat(event.Name)
			if err == nil || os.IsExist(err) {
				w.fw.Add(event.Name)
			}
		}

		c, err := w.f.Read()
		if err != nil {
			return nil, err
		}

		// add path again for the event bug of fsnotify
		w.fw.Add(w.f.path)

		return c, nil
	case err, ok := <-w.fw.Errors:
		// check if channel was closed (i.e. Watcher.Close() was called).
		if !ok {
			return nil, source.ErrWatcherStopped
		}

		return nil, err
	}
}

func (w *watcher) Stop() error {
	return w.fw.Close()
}
