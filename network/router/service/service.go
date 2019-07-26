package service

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/network/router"
	pb "github.com/micro/go-micro/network/router/proto"
)

var (
	// ErrNotImplemented means the functionality has not been implemented
	ErrNotImplemented = errors.New("not implemented")
)

type svc struct {
	opts     router.Options
	router   pb.RouterService
	status   router.Status
	watchers map[string]*svcWatcher
	sync.RWMutex
}

// NewRouter creates new service router and returns it
func NewRouter(opts ...router.Option) router.Router {
	// get default options
	options := router.DefaultOptions()

	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	// NOTE: might need some client opts here
	client := client.DefaultClient

	// NOTE: should we have Client/Service option in router.Options?
	s := &svc{
		opts:     options,
		router:   pb.NewRouterService(router.DefaultName, client),
		status:   router.Status{Code: router.Stopped, Error: nil},
		watchers: make(map[string]*svcWatcher),
	}

	return s
}

// Init initializes router with given options
func (s *svc) Init(opts ...router.Option) error {
	for _, o := range opts {
		o(&s.opts)
	}
	return nil
}

// Options returns router options
func (s *svc) Options() router.Options {
	return s.opts
}

// Run runs the router.
// It returns error if the router is already running.
func (s *svc) run() {
	s.Lock()
	defer s.Unlock()

	switch s.status.Code {
	case router.Stopped, router.Error:
		// TODO: start event stream watcher
		// TODO: start watchError monitor
	}
}

// Advertise advertises routes to the network
func (s *svc) Advertise() (<-chan *router.Advert, error) {
	// TODO: start advert stream watcher
	return nil, nil
}

// Process processes incoming adverts
func (s *svc) Process(a *router.Advert) error {
	return nil
}

// Create new route in the routing table
func (s *svc) Create(r router.Route) error {
	route := &pb.Route{
		Service: r.Service,
		Address: r.Address,
		Gateway: r.Gateway,
		Network: r.Network,
		Link:    r.Link,
		Metric:  int64(r.Metric),
	}

	if _, err := s.router.Create(context.Background(), route); err != nil {
		return err
	}

	return nil
}

// Delete deletes existing route from the routing table
func (s *svc) Delete(r router.Route) error {
	route := &pb.Route{
		Service: r.Service,
		Address: r.Address,
		Gateway: r.Gateway,
		Network: r.Network,
		Link:    r.Link,
		Metric:  int64(r.Metric),
	}

	if _, err := s.router.Delete(context.Background(), route); err != nil {
		return err
	}

	return nil
}

// Update updates route in the routing table
func (s *svc) Update(r router.Route) error {
	route := &pb.Route{
		Service: r.Service,
		Address: r.Address,
		Gateway: r.Gateway,
		Network: r.Network,
		Link:    r.Link,
		Metric:  int64(r.Metric),
	}

	if _, err := s.router.Update(context.Background(), route); err != nil {
		return err
	}

	return nil
}

// List returns the list of all routes in the table
func (s *svc) List() ([]router.Route, error) {
	resp, err := s.router.List(context.Background(), &pb.ListRequest{})
	if err != nil {
		return nil, err
	}

	routes := make([]router.Route, len(resp.Routes))
	for i, route := range resp.Routes {
		routes[i] = router.Route{
			Service: route.Service,
			Address: route.Address,
			Gateway: route.Gateway,
			Network: route.Network,
			Link:    route.Link,
			Metric:  int(route.Metric),
		}
	}

	return routes, nil
}

// Lookup looks up routes in the routing table and returns them
func (s *svc) Lookup(q router.Query) ([]router.Route, error) {
	// call the router
	resp, err := s.router.Lookup(context.Background(), &pb.LookupRequest{
		Query: &pb.Query{
			Service: q.Options().Service,
			Gateway: q.Options().Gateway,
			Network: q.Options().Network,
		},
	})

	// errored out
	if err != nil {
		return nil, err
	}

	routes := make([]router.Route, len(resp.Routes))
	for i, route := range resp.Routes {
		routes[i] = router.Route{
			Service: route.Service,
			Address: route.Address,
			Gateway: route.Gateway,
			Network: route.Network,
			Link:    route.Link,
			Metric:  int(route.Metric),
		}
	}

	return routes, nil
}

// Watch returns a watcher which allows to track updates to the routing table
func (s *svc) Watch(opts ...router.WatchOption) (router.Watcher, error) {
	wopts := router.WatchOptions{
		Service: "*",
	}

	for _, o := range opts {
		o(&wopts)
	}

	w := &svcWatcher{
		opts:    wopts,
		resChan: make(chan *router.Event, 10),
		done:    make(chan struct{}),
	}

	s.Lock()
	s.watchers[uuid.New().String()] = w
	s.Unlock()

	return w, nil
}

// Status returns router status
func (s *svc) Status() router.Status {
	s.RLock()
	defer s.RUnlock()

	// make a copy of the status
	status := s.status

	return status
}

// Stop stops the router
func (s *svc) Stop() error {
	return nil
}

// Returns the router implementation
func (s *svc) String() string {
	return "service"
}
