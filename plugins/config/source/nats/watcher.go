package nats

import (
	"time"

	natsgo "github.com/nats-io/nats.go"
	"go-micro.dev/v4/config/encoder"
	"go-micro.dev/v4/config/source"
)

type watcher struct {
	e      encoder.Encoder
	name   string
	bucket string
	key    string

	ch   chan *source.ChangeSet
	exit chan bool
}

func newWatcher(kv natsgo.KeyValue, bucket, key, name string, e encoder.Encoder) (source.Watcher, error) {
	w := &watcher{
		e:      e,
		name:   name,
		bucket: bucket,
		key:    key,
		ch:     make(chan *source.ChangeSet),
		exit:   make(chan bool),
	}

	wh, _ := kv.Watch(key)

	go func() {
		for {
			select {
			case v := <-wh.Updates():
				if v != nil {
					w.handle(v.Value())
				}
			case <-w.exit:
				_ = wh.Stop()
				return
			}
		}
	}()
	return w, nil
}

func (w *watcher) handle(data []byte) {
	cs := &source.ChangeSet{
		Timestamp: time.Now(),
		Format:    w.e.String(),
		Source:    w.name,
		Data:      data,
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
		close(w.exit)
	}

	return nil
}
