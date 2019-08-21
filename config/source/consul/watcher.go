package consul

import (
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/micro/go-micro/config/encoder"
	"github.com/micro/go-micro/config/source"
)

type watcher struct {
	e           encoder.Encoder
	name        string
	stripPrefix string

	wp   *watch.Plan
	ch   chan *source.ChangeSet
	exit chan bool
}

func newWatcher(key, addr, name, stripPrefix string, e encoder.Encoder) (source.Watcher, error) {
	w := &watcher{
		e:           e,
		name:        name,
		stripPrefix: stripPrefix,
		ch:          make(chan *source.ChangeSet),
		exit:        make(chan bool),
	}

	wp, err := watch.Parse(map[string]interface{}{"type": "keyprefix", "prefix": key})
	if err != nil {
		return nil, err
	}

	wp.Handler = w.handle

	// wp.Run is a blocking call and will prevent newWatcher from returning
	go wp.Run(addr)

	w.wp = wp

	return w, nil
}

func (w *watcher) handle(idx uint64, data interface{}) {
	if data == nil {
		return
	}

	kvs, ok := data.(api.KVPairs)
	if !ok {
		return
	}

	d, err := makeMap(w.e, kvs, w.stripPrefix)
	if err != nil {
		return
	}

	b, err := w.e.Encode(d)
	if err != nil {
		return
	}

	cs := &source.ChangeSet{
		Timestamp: time.Now(),
		Format:    w.e.String(),
		Source:    w.name,
		Data:      b,
	}
	cs.Checksum = cs.Sum()

	w.ch <- cs
}

func (w *watcher) Next() (*source.ChangeSet, error) {
	select {
	case cs := <-w.ch:
		return cs, nil
	case <-w.exit:
		return nil, source.ErrWatcherStopped
	}
}

func (w *watcher) Stop() error {
	select {
	case <-w.exit:
		return nil
	default:
		w.wp.Stop()
		close(w.exit)
	}
	return nil
}
