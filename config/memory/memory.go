package memory

import (
	"bytes"
	"sync"
	"time"

	"github.com/asim/nitro/v3/config"
	"github.com/asim/nitro/v3/config/loader"
	"github.com/asim/nitro/v3/config/loader/memory"
	"github.com/asim/nitro/v3/config/reader"
	"github.com/asim/nitro/v3/config/reader/json"
)

type memoryConfig struct {
	exit chan bool
	opts config.Options

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

func NewConfig(opts ...config.Option) (config.Config, error) {
	var c memoryConfig

	if err := c.Init(opts...); err != nil {
		return nil, err
	}

	go c.run()
	return &c, nil
}

func (c *memoryConfig) Init(opts ...config.Option) error {
	c.opts = config.Options{
		Reader: json.NewReader(),
	}
	c.exit = make(chan bool)
	for _, o := range opts {
		o(&c.opts)
	}

	// default loader uses the configured reader
	if c.opts.Loader == nil {
		c.opts.Loader = memory.NewLoader(memory.WithReader(c.opts.Reader))
	}

	err := c.opts.Loader.Load(c.opts.Source...)
	if err != nil {
		return err
	}

	c.snap, err = c.opts.Loader.Snapshot()
	if err != nil {
		return err
	}

	c.vals, err = c.opts.Reader.Values(c.snap.ChangeSet)
	if err != nil {
		return err
	}

	return nil
}

func (c *memoryConfig) Options() config.Options {
	return c.opts
}

func (c *memoryConfig) run() {
	watch := func(w loader.Watcher) error {
		for {
			// get changeset
			snap, err := w.Next()
			if err != nil {
				return err
			}

			c.Lock()

			if c.snap != nil && c.snap.Version >= snap.Version {
				c.Unlock()
				continue
			}

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

// sync loads all the sources, calls the parser and updates the config
func (c *memoryConfig) Sync() error {
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

func (c *memoryConfig) Close() error {
	select {
	case <-c.exit:
		return nil
	default:
		close(c.exit)
	}
	return nil
}

func (c *memoryConfig) Get(path ...string) reader.Value {
	c.RLock()
	defer c.RUnlock()

	// did sync actually work?
	if c.vals != nil {
		return c.vals.Get(path...)
	}

	// no value
	return newValue()
}

func (c *memoryConfig) Load(path ...string) (reader.Value, error) {
	return c.Get(path...), nil
}

func (c *memoryConfig) Watch(path ...string) (config.Watcher, error) {
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

func (c *memoryConfig) String() string {
	return "memory"
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
