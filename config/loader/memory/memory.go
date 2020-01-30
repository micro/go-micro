package memory

import (
	"bytes"
	"container/list"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-micro/v2/config/loader"
	"github.com/micro/go-micro/v2/config/reader"
	"github.com/micro/go-micro/v2/config/reader/json"
	"github.com/micro/go-micro/v2/config/source"
)

type memory struct {
	exit chan bool
	opts loader.Options

	sync.RWMutex
	// the current snapshot
	snap *loader.Snapshot
	// the current values
	vals reader.Values
	// all the sets
	sets []*source.ChangeSet
	// all the sources
	sources []source.Source

	watchers *list.List
}

type watcher struct {
	exit    chan bool
	path    []string
	value   reader.Value
	reader  reader.Reader
	updates chan reader.Value
}

func (m *memory) watch(idx int, s source.Source) {
	m.Lock()
	m.sets = append(m.sets, &source.ChangeSet{Source: s.String()})
	m.Unlock()

	// watches a source for changes
	watch := func(idx int, s source.Watcher) error {
		for {
			// get changeset
			cs, err := s.Next()
			if err != nil {
				return err
			}

			m.Lock()

			// save
			m.sets[idx] = cs

			// merge sets
			set, err := m.opts.Reader.Merge(m.sets...)
			if err != nil {
				m.Unlock()
				return err
			}

			// set values
			m.vals, _ = m.opts.Reader.Values(set)
			m.snap = &loader.Snapshot{
				ChangeSet: set,
				Version:   fmt.Sprintf("%d", time.Now().Unix()),
			}
			m.Unlock()

			// send watch updates
			m.update()
		}
	}

	for {
		// watch the source
		w, err := s.Watch()
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		done := make(chan bool)

		// the stop watch func
		go func() {
			select {
			case <-done:
			case <-m.exit:
			}
			w.Stop()
		}()

		// block watch
		if err := watch(idx, w); err != nil {
			// do something better
			time.Sleep(time.Second)
		}

		// close done chan
		close(done)

		// if the config is closed exit
		select {
		case <-m.exit:
			return
		default:
		}
	}
}

func (m *memory) loaded() bool {
	var loaded bool
	m.RLock()
	if m.vals != nil {
		loaded = true
	}
	m.RUnlock()
	return loaded
}

// reload reads the sets and creates new values
func (m *memory) reload() error {
	m.Lock()

	// merge sets
	set, err := m.opts.Reader.Merge(m.sets...)
	if err != nil {
		m.Unlock()
		return err
	}

	// set values
	m.vals, _ = m.opts.Reader.Values(set)
	m.snap = &loader.Snapshot{
		ChangeSet: set,
		Version:   fmt.Sprintf("%d", time.Now().Unix()),
	}

	m.Unlock()

	// update watchers
	m.update()

	return nil
}

func (m *memory) update() {
	watchers := make([]*watcher, 0, m.watchers.Len())

	m.RLock()
	for e := m.watchers.Front(); e != nil; e = e.Next() {
		watchers = append(watchers, e.Value.(*watcher))
	}
	m.RUnlock()

	for _, w := range watchers {
		select {
		case w.updates <- m.vals.Get(w.path...):
		default:
		}
	}
}

// Snapshot returns a snapshot of the current loaded config
func (m *memory) Snapshot() (*loader.Snapshot, error) {
	if m.loaded() {
		m.RLock()
		snap := loader.Copy(m.snap)
		m.RUnlock()
		return snap, nil
	}

	// not loaded, sync
	if err := m.Sync(); err != nil {
		return nil, err
	}

	// make copy
	m.RLock()
	snap := loader.Copy(m.snap)
	m.RUnlock()

	return snap, nil
}

// Sync loads all the sources, calls the parser and updates the config
func (m *memory) Sync() error {
	//nolint:prealloc
	var sets []*source.ChangeSet

	m.Lock()

	// read the source
	var gerr []string

	for _, source := range m.sources {
		ch, err := source.Read()
		if err != nil {
			gerr = append(gerr, err.Error())
			continue
		}
		sets = append(sets, ch)
	}

	// merge sets
	set, err := m.opts.Reader.Merge(sets...)
	if err != nil {
		m.Unlock()
		return err
	}

	// set values
	vals, err := m.opts.Reader.Values(set)
	if err != nil {
		m.Unlock()
		return err
	}
	m.vals = vals
	m.snap = &loader.Snapshot{
		ChangeSet: set,
		Version:   fmt.Sprintf("%d", time.Now().Unix()),
	}

	m.Unlock()

	// update watchers
	m.update()

	if len(gerr) > 0 {
		return fmt.Errorf("source loading errors: %s", strings.Join(gerr, "\n"))
	}

	return nil
}

func (m *memory) Close() error {
	select {
	case <-m.exit:
		return nil
	default:
		close(m.exit)
	}
	return nil
}

func (m *memory) Get(path ...string) (reader.Value, error) {
	if !m.loaded() {
		if err := m.Sync(); err != nil {
			return nil, err
		}
	}

	m.Lock()
	defer m.Unlock()

	// did sync actually work?
	if m.vals != nil {
		return m.vals.Get(path...), nil
	}

	// assuming vals is nil
	// create new vals

	ch := m.snap.ChangeSet

	// we are truly screwed, trying to load in a hacked way
	v, err := m.opts.Reader.Values(ch)
	if err != nil {
		return nil, err
	}

	// lets set it just because
	m.vals = v

	if m.vals != nil {
		return m.vals.Get(path...), nil
	}

	// ok we're going hardcore now
	return nil, errors.New("no values")
}

func (m *memory) Load(sources ...source.Source) error {
	var gerrors []string

	for _, source := range sources {
		set, err := source.Read()
		if err != nil {
			gerrors = append(gerrors,
				fmt.Sprintf("error loading source %s: %v",
					source,
					err))
			// continue processing
			continue
		}
		m.Lock()
		m.sources = append(m.sources, source)
		m.sets = append(m.sets, set)
		idx := len(m.sets) - 1
		m.Unlock()
		go m.watch(idx, source)
	}

	if err := m.reload(); err != nil {
		gerrors = append(gerrors, err.Error())
	}

	// Return errors
	if len(gerrors) != 0 {
		return errors.New(strings.Join(gerrors, "\n"))
	}
	return nil
}

func (m *memory) Watch(path ...string) (loader.Watcher, error) {
	value, err := m.Get(path...)
	if err != nil {
		return nil, err
	}

	m.Lock()

	w := &watcher{
		exit:    make(chan bool),
		path:    path,
		value:   value,
		reader:  m.opts.Reader,
		updates: make(chan reader.Value, 1),
	}

	e := m.watchers.PushBack(w)

	m.Unlock()

	go func() {
		<-w.exit
		m.Lock()
		m.watchers.Remove(e)
		m.Unlock()
	}()

	return w, nil
}

func (m *memory) String() string {
	return "memory"
}

func (w *watcher) Next() (*loader.Snapshot, error) {
	for {
		select {
		case <-w.exit:
			return nil, errors.New("watcher stopped")
		case v := <-w.updates:
			if bytes.Equal(w.value.Bytes(), v.Bytes()) {
				continue
			}
			w.value = v

			cs := &source.ChangeSet{
				Data:      v.Bytes(),
				Format:    w.reader.String(),
				Source:    "memory",
				Timestamp: time.Now(),
			}
			cs.Sum()

			return &loader.Snapshot{
				ChangeSet: cs,
				Version:   fmt.Sprintf("%d", time.Now().Unix()),
			}, nil
		}
	}
}

func (w *watcher) Stop() error {
	select {
	case <-w.exit:
	default:
		close(w.exit)
	}
	return nil
}

func NewLoader(opts ...loader.Option) loader.Loader {
	options := loader.Options{
		Reader: json.NewReader(),
	}

	for _, o := range opts {
		o(&options)
	}

	m := &memory{
		exit:     make(chan bool),
		opts:     options,
		watchers: list.New(),
		sources:  options.Source,
	}

	for i, s := range options.Source {
		go m.watch(i, s)
	}

	return m
}
