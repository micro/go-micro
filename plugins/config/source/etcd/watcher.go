package etcd

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/asim/go-micro/v3/config/source"
	"go.etcd.io/etcd/client/v3"
)

type watcher struct {
	opts        source.Options
	name        string
	stripPrefix string

	sync.RWMutex
	cs *source.ChangeSet

	ch   chan *source.ChangeSet
	exit chan bool
}

func newWatcher(key, strip string, wc clientv3.Watcher, cs *source.ChangeSet, opts source.Options) (source.Watcher, error) {
	w := &watcher{
		opts:        opts,
		name:        "etcd",
		stripPrefix: strip,
		cs:          cs,
		ch:          make(chan *source.ChangeSet),
		exit:        make(chan bool),
	}

	ch := wc.Watch(context.Background(), key, clientv3.WithPrefix())

	go w.run(wc, ch)

	return w, nil
}

func (w *watcher) handle(evs []*clientv3.Event) {
	w.RLock()
	data := w.cs.Data
	w.RUnlock()

	var vals map[string]interface{}

	// unpackage existing changeset
	if err := w.opts.Encoder.Decode(data, &vals); err != nil {
		return
	}

	// update base changeset
	d := makeEvMap(w.opts.Encoder, vals, evs, w.stripPrefix)

	// pack the changeset
	b, err := w.opts.Encoder.Encode(d)
	if err != nil {
		return
	}

	// create new changeset
	cs := &source.ChangeSet{
		Timestamp: time.Now(),
		Source:    w.name,
		Data:      b,
		Format:    w.opts.Encoder.String(),
	}
	cs.Checksum = cs.Sum()

	// set base change set
	w.Lock()
	w.cs = cs
	w.Unlock()

	// send update
	w.ch <- cs
}

func (w *watcher) run(wc clientv3.Watcher, ch clientv3.WatchChan) {
	for {
		select {
		case rsp, ok := <-ch:
			if !ok {
				return
			}
			w.handle(rsp.Events)
		case <-w.exit:
			wc.Close()
			return
		}
	}
}

func (w *watcher) Next() (*source.ChangeSet, error) {
	select {
	case cs := <-w.ch:
		return cs, nil
	case <-w.exit:
		return nil, errors.New("watcher stopped")
	}
}

func (w *watcher) Stop() error {
	select {
	case <-w.exit:
		return nil
	default:
		close(w.exit)
	}
	return nil
}
