package service

import (
	"sync"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/network/router"
	pb "github.com/micro/go-micro/network/router/proto"
)

type svc struct {
	router pb.RouterService
	opts   router.Options
	status router.Status
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
		opts:   options,
		status: router.Status{Code: router.Stopped, Error: nil},
		router: pb.NewRouterService(router.DefaultName, client),
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

// Advertise advertises routes to the network
func (s *svc) Advertise() (<-chan *router.Advert, error) {
	return nil, nil
}

// Process processes incoming adverts
func (s *svc) Process(a *router.Advert) error {
	return nil
}

// Create new route in the routing table
func (s *svc) Create(r router.Route) error {
	return nil
}

// Delete deletes existing route from the routing table
func (s *svc) Delete(r router.Route) error {
	return nil
}

// Update updates route in the routing table
func (s *svc) Update(r router.Route) error {
	return nil
}

// List returns the list of all routes in the table
func (s *svc) List() ([]router.Route, error) {
	return nil, nil
}

// Lookup looks up routes in the routing table and returns them
func (s *svc) Lookup(q router.Query) ([]router.Route, error) {
	return nil, nil
}

// Watch returns a watcher which allows to track updates to the routing table
func (s *svc) Watch(opts ...router.WatchOption) (router.Watcher, error) {
	return nil, nil
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
