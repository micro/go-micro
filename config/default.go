package config

import (
	"bytes"
	"sync"
	"time"

	"github.com/micro/go-micro/config/loader"
	"github.com/micro/go-micro/config/loader/memory"
	"github.com/micro/go-micro/config/reader"
	"github.com/micro/go-micro/config/reader/json"
	"github.com/micro/go-micro/config/source"
)

type config struct {
	exit chan bool
	opts Options

	sync.RWMutex
	// the current snapshot
	snap *loader.Snapshot
	// the current values
	vals reader.Values
}

type watcher struct {
	lw    loader.Watcher
	rd    reader.Reader
	path  []string
	value reader.Value
}

func newConfig(opts ...Option) Config {
	options := Options{
		Loader: memory.NewLoader(),
		Reader: json.NewReader(),
	}

	for _, o := range opts {
		o(&options)
	}

	options.Loader.Load(options.Source...)
	snap, _ := options.Loader.Snapshot()
	vals, _ := options.Reader.Values(snap.ChangeSet)

	c := &config{
		exit: make(chan bool),
		opts: options,
		snap: snap,
		vals: vals,
	}

	go c.run()

	return c
}

func (c *config) run() {
	watch := func(w loader.Watcher) error {
		for {
			// get changeset
			snap, err := w.Next()
			if err != nil {
				return err
			}

			c.Lock()

			// save
			c.snap = snap

			// set values
			c.vals, _ = c.opts.Reader.Values(snap.ChangeSet)

			c.Unlock()
		}
	}

	for {
		w, err := c.opts.Loader.Watch()
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		done := make(chan bool)

		// the stop watch func
		go func() {
			select {
			case <-done:
			case <-c.exit:
			}
			w.Stop()
		}()

		// block watch
		if err := watch(w); err != nil {
			// do something better
			time.Sleep(time.Second)
		}

		// close done chan
		close(done)

		// if the config is closed exit
		select {
		case <-c.exit:
			return
		default:
		}
	}
}

func (c *config) Map() map[string]interface{} {
	c.RLock()
	defer c.RUnlock()
	return c.vals.Map()
}

func (c *config) Scan(v interface{}) error {
	c.RLock()
	defer c.RUnlock()
	return c.vals.Scan(v)
}

// sync loads all the sources, calls the parser and updates the config
func (c *config) Sync() error {
	if err := c.opts.Loader.Sync(); err != nil {
		return err
	}

	snap, err := c.opts.Loader.Snapshot()
	if err != nil {
		return err
	}

	c.Lock()
	defer c.Unlock()

	c.snap = snap
	vals, err := c.opts.Reader.Values(snap.ChangeSet)
	if err != nil {
		return err
	}
	c.vals = vals

	return nil
}

func (c *config) Close() error {
	select {
	case <-c.exit:
		return nil
	default:
		close(c.exit)
	}
	return nil
}

func (c *config) Get(path ...string) reader.Value {
	c.RLock()
	defer c.RUnlock()

	// did sync actually work?
	if c.vals != nil {
		return c.vals.Get(path...)
	}

	// no value
	return newValue()
}

func (c *config) Bytes() []byte {
	c.RLock()
	defer c.RUnlock()

	if c.vals == nil {
		return []byte{}
	}

	return c.vals.Bytes()
}

func (c *config) Load(sources ...source.Source) error {
	if err := c.opts.Loader.Load(sources...); err != nil {
		return err
	}

	snap, err := c.opts.Loader.Snapshot()
	if err != nil {
		return err
	}

	c.Lock()
	defer c.Unlock()

	c.snap = snap
	vals, err := c.opts.Reader.Values(snap.ChangeSet)
	if err != nil {
		return err
	}
	c.vals = vals

	return nil
}

func (c *config) Watch(path ...string) (Watcher, error) {
	value := c.Get(path...)

	w, err := c.opts.Loader.Watch(path...)
	if err != nil {
		return nil, err
	}

	return &watcher{
		lw:    w,
		rd:    c.opts.Reader,
		path:  path,
		value: value,
	}, nil
}

func (c *config) String() string {
	return "config"
}

func (w *watcher) Next() (reader.Value, error) {
	for {
		s, err := w.lw.Next()
		if err != nil {
			return nil, err
		}

		// only process changes
		if bytes.Equal(w.value.Bytes(), s.ChangeSet.Data) {
			continue
		}

		v, err := w.rd.Values(s.ChangeSet)
		if err != nil {
			return nil, err
		}

		w.value = v.Get()
		return w.value, nil
	}
}

func (w *watcher) Stop() error {
	return w.lw.Stop()
}
