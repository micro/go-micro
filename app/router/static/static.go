// Package static is a static router which returns the service name as the address + port
package static

import (
	"github.com/asim/nitro/app/router"
)

var (
	// DefaulPort is the port to append where nothing is set
	DefaultPort = 8080
)

// NewRouter returns an initialized static router
func NewRouter(opts ...router.Option) router.Router {
	options := router.DefaultOptions()
	for _, o := range opts {
		o(&options)
	}
	return &static{options}
}

type static struct {
	options router.Options
}

func (s *static) Init(opts ...router.Option) error {
	for _, o := range opts {
		o(&s.options)
	}
	return nil
}

func (s *static) Options() router.Options {
	return s.options
}

func (s *static) Table() router.Table {
	return nil
}

func (s *static) Lookup(service string, opts ...router.LookupOption) ([]router.Route, error) {
	options := router.NewLookup(opts...)

	return []router.Route{
		router.Route{
			Service: service,
			Address: service,
			Gateway: options.Gateway,
			Network: options.Network,
			Router:  options.Router,
		},
	}, nil
}

// Watch will return a noop watcher
func (s *static) Watch(opts ...router.WatchOption) (router.Watcher, error) {
	return &watcher{
		events: make(chan *router.Event),
	}, nil
}

func (s *static) Close() error {
	return nil
}

func (s *static) String() string {
	return "static"
}

// watcher is a noop implementation
type watcher struct {
	events chan *router.Event
}

// Next is a blocking call that returns watch result
func (w *watcher) Next() (*router.Event, error) {
	e := <-w.events
	return e, nil
}

// Chan returns event channel
func (w *watcher) Chan() (<-chan *router.Event, error) {
	return w.events, nil
}

// Stop stops watcher
func (w *watcher) Stop() {
	return
}
