package multi

import (
	"context"
	"sync"

	log "github.com/asim/go-micro/v3/logger"
	"github.com/asim/go-micro/v3/registry"
)

type multiRegistry struct {
	r    []registry.Registry
	w    []registry.Registry
	opts registry.Options
}

func (m *multiRegistry) Init(opts ...registry.Option) error {
	return configure(m, opts...)
}

func (m *multiRegistry) Options() registry.Options {
	return m.opts
}

func (m *multiRegistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	var wg sync.WaitGroup
	var errs []error

	done := make(chan bool)
	cerr := make(chan error)

	wg.Add(len(m.w))

	go func() {
		for {
			select {
			case <-done:
				return
			case err := <-cerr:
				errs = append(errs, err)
				wg.Done()
			}
		}
	}()

	for _, mw := range m.w {
		go func(w registry.Registry) {
			if err := w.Register(s, opts...); err != nil {
				cerr <- err
			} else {
				wg.Done()
			}
		}(mw)
	}

	wg.Wait()
	defer close(done)

	if len(errs) > 0 {
		return errs[0]
	}

	return nil

}

func (m *multiRegistry) Deregister(s *registry.Service, opts ...registry.DeregisterOption) error {
	var wg sync.WaitGroup
	var errs []error

	done := make(chan bool)
	cerr := make(chan error)

	wg.Add(len(m.w))

	go func() {
		for {
			select {
			case <-done:
				return
			case err := <-cerr:
				errs = append(errs, err)
				wg.Done()
			}
		}
	}()

	for _, mw := range m.w {
		go func(w registry.Registry) {
			if err := w.Deregister(s); err != nil {
				cerr <- err
			} else {
				wg.Done()
			}
		}(mw)
	}

	wg.Wait()
	defer close(done)

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func (m *multiRegistry) GetService(n string, opts ...registry.GetOption) ([]*registry.Service, error) {
	var wg sync.WaitGroup
	var errs []error
	var svcs []*registry.Service
	var mu sync.Mutex

	done := make(chan bool)
	cerr := make(chan error)
	csvc := make(chan []*registry.Service)

	wg.Add(len(m.r))

	go func() {
		for {
			select {
			case <-done:
				return
			case err := <-cerr:
				errs = append(errs, err)
				wg.Done()
			case svc := <-csvc:
				mu.Lock()
				svcs = append(svcs, svc...)
				mu.Unlock()
				wg.Done()
			}
		}
	}()

	for _, mr := range m.r {
		go func(r registry.Registry) {
			svc, err := r.GetService(n)
			if err != nil && err != registry.ErrNotFound {
				cerr <- err
			} else {
				csvc <- svc
			}
		}(mr)
	}

	wg.Wait()
	defer close(done)

	if len(errs) > 0 {
		return nil, errs[0]
	}

	mu.Lock()
	if len(svcs) == 0 {
		return nil, registry.ErrNotFound
	}
	mu.Unlock()

	return svcs, nil
}

func (m *multiRegistry) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	var wg sync.WaitGroup
	var errs []error
	var svcs []*registry.Service
	var mu sync.Mutex

	done := make(chan bool)
	cerr := make(chan error)
	csvc := make(chan []*registry.Service)

	wg.Add(len(m.r))

	go func() {
		for {
			select {
			case <-done:
				return
			case err := <-cerr:
				errs = append(errs, err)
				wg.Done()
			case svc := <-csvc:
				mu.Lock()
				svcs = append(svcs, svc...)
				mu.Unlock()
				wg.Done()
			}
		}
	}()

	for _, mr := range m.r {
		go func(r registry.Registry) {
			if svc, err := r.ListServices(); err != nil {
				cerr <- err
			} else {
				csvc <- svc
			}
		}(mr)
	}

	wg.Wait()
	defer close(done)

	if len(errs) > 0 {
		return nil, errs[0]
	}

	mu.Lock()
	ret := svcs
	mu.Unlock()
	return ret, nil
}

func (m *multiRegistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	return newMultiWatcher(m.r, opts...)
}

func (m *multiRegistry) String() string {
	return "multi"
}

func NewRegistry(opts ...registry.Option) registry.Registry {
	m := &multiRegistry{
		opts: registry.Options{
			Context: context.Background(),
		},
	}

	if err := configure(m, opts...); err != nil {
		log.Fatalf("[multi] Error configuring registry: %v", err)
	}

	return m
}

func configure(m *multiRegistry, opts ...registry.Option) error {
	// parse options
	for _, o := range opts {
		o(&m.opts)
	}

	if w, ok := m.opts.Context.Value(writeKey{}).([]registry.Registry); ok && w != nil {
		m.w = w
	}

	m.r = m.w

	if r, ok := m.opts.Context.Value(readKey{}).([]registry.Registry); ok && r != nil {
		m.r = append(m.r, r...)
	}
	return nil
}
