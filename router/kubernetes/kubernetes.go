// Package kubernetes is a kubernetes router which uses the service name and network to route
package kubernetes

import (
	"fmt"

	"github.com/micro/go-micro/v3/router"
)

// NewRouter returns an initialized kubernetes router
func NewRouter(opts ...router.Option) router.Router {
	options := router.DefaultOptions()
	for _, o := range opts {
		o(&options)
	}
	return &kubernetes{options}
}

type kubernetes struct {
	options router.Options
}

func (k *kubernetes) Init(opts ...router.Option) error {
	for _, o := range opts {
		o(&k.options)
	}
	return nil
}

func (k *kubernetes) Options() router.Options {
	return k.options
}

func (k *kubernetes) Table() router.Table {
	return new(table)
}

func (k *kubernetes) Lookup(service string, opts ...router.LookupOption) ([]router.Route, error) {
	options := router.NewLookup(opts...)
	if len(options.Network) == 0 {
		options.Network = "micro"
	}

	address := fmt.Sprintf("%v.%v.svc.cluster.local:8080", service, options.Network)

	return []router.Route{
		router.Route{
			Service: service,
			Address: address,
			Gateway: options.Gateway,
			Network: options.Network,
			Router:  options.Router,
		},
	}, nil
}

// Watch will return a noop watcher
func (k *kubernetes) Watch(opts ...router.WatchOption) (router.Watcher, error) {
	return &watcher{
		events: make(chan *router.Event),
	}, nil
}

func (k *kubernetes) Close() error {
	return nil
}

func (k *kubernetes) String() string {
	return "kubernetes"
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

type table struct{}

// Create new route in the routing table
func (t *table) Create(router.Route) error {
	return nil
}

// Delete existing route from the routing table
func (t *table) Delete(router.Route) error {
	return nil
}

// Update route in the routing table
func (t *table) Update(router.Route) error {
	return nil
}

// Read is for querying the table
func (t *table) Read(...router.ReadOption) ([]router.Route, error) {
	return []router.Route{}, nil
}
