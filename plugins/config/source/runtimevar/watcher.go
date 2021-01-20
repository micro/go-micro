package runtimevar

import (
	"context"
	"errors"
	"fmt"

	"github.com/asim/go-micro/v3/config/source"
	"gocloud.dev/runtimevar"
)

type watcher struct {
	name string
	opts source.Options
	exit chan bool
	v    *runtimevar.Variable
}

func (w *watcher) Next() (*source.ChangeSet, error) {
	// check exit status
	select {
	case <-w.exit:
		return nil, errors.New("watcher stopped")
	default:
	}

	// create a context with cancellation to kill watcher
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		// if the watcher stops then cancel
		select {
		case <-w.exit:
			cancel()
		}
	}()

	s, err := w.v.Watch(ctx)
	if err != nil {
		return nil, err
	}

	// assuming value is bytes
	b, err := w.opts.Encoder.Encode(s.Value.([]byte))
	if err != nil {
		return nil, fmt.Errorf("error reading source: %v", err)
	}

	cs := &source.ChangeSet{
		Timestamp: s.UpdateTime,
		Format:    w.opts.Encoder.String(),
		Source:    w.name,
		Data:      b,
	}
	cs.Checksum = cs.Sum()

	return cs, nil
}

func (w *watcher) Stop() error {
	select {
	case <-w.exit:
		return nil
	default:
		close(w.exit)
		return nil
	}
}

func newWatcher(name string, v *runtimevar.Variable, opts source.Options) (source.Watcher, error) {
	return &watcher{
		name: name,
		opts: opts,
		exit: make(chan bool),
		v:    v,
	}, nil
}
