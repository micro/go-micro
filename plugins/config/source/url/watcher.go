package url

import (
	"errors"

	"github.com/asim/go-micro/v3/config/source"
)

type urlWatcher struct {
	u    *urlSource
	exit chan bool
}

func newWatcher(u *urlSource) (*urlWatcher, error) {
	return &urlWatcher{
		u:    u,
		exit: make(chan bool),
	}, nil
}

func (u *urlWatcher) Next() (*source.ChangeSet, error) {
	<-u.exit
	return nil, errors.New("url watcher stopped")
}

func (u *urlWatcher) Stop() error {
	select {
	case <-u.exit:
	default:
	}
	return nil
}
