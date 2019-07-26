package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

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
	opts       router.Options
	router     pb.RouterService
	status     router.Status
	watchers   map[string]*svcWatcher
	exit       chan struct{}
	errChan    chan error
	advertChan chan *router.Advert
	wg         *sync.WaitGroup
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
		wg:       &sync.WaitGroup{},
	}

	go s.run()

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

// watchErrors watches router errors and takes appropriate actions
func (s *svc) watchErrors() {
	var err error

	select {
	case <-s.exit:
	case err = <-s.errChan:
	}

	s.Lock()
	defer s.Unlock()
	if s.status.Code != router.Stopped {
		// notify all goroutines to finish
		close(s.exit)
		// TODO" might need to drain some channels here
	}

	if err != nil {
		s.status = router.Status{Code: router.Error, Error: err}
	}
}

// watchRouter watches router and send events to all registered watchers
func (s *svc) watchRouter(stream pb.Router_WatchService) error {
	defer stream.Close()
	var watchErr error

	for {
		resp, err := stream.Recv()
		if err != nil {
			if err != io.EOF {
				watchErr = err
			}
			break
		}

		route := router.Route{
			Service: resp.Route.Service,
			Address: resp.Route.Address,
			Gateway: resp.Route.Gateway,
			Network: resp.Route.Network,
			Link:    resp.Route.Link,
			Metric:  int(resp.Route.Metric),
		}

		event := &router.Event{
			Type:      router.EventType(resp.Type),
			Timestamp: time.Unix(0, resp.Timestamp),
			Route:     route,
		}

		s.RLock()
		for _, w := range s.watchers {
			select {
			case w.resChan <- event:
			case <-w.done:
			}
		}
		s.RUnlock()
	}

	return watchErr
}

// Run runs the router.
// It returns error if the router is already running.
func (s *svc) run() {
	s.Lock()
	defer s.Unlock()

	switch s.status.Code {
	case router.Stopped, router.Error:
		stream, err := s.router.Watch(context.Background(), &pb.WatchRequest{})
		if err != nil {
			s.status = router.Status{Code: router.Error, Error: fmt.Errorf("failed getting router stream: %s", err)}
			return
		}

		// create error and exit channels
		s.errChan = make(chan error, 1)
		s.exit = make(chan struct{})

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			select {
			case s.errChan <- s.watchRouter(stream):
			case <-s.exit:
			}
		}()

		// watch for errors and cleanup
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.watchErrors()
		}()

		// mark router as Running and set its Error to nil
		s.status = router.Status{Code: router.Running, Error: nil}

		return
	}

	return
}

func (s *svc) advertiseEvents(stream pb.Router_AdvertiseService) error {
	defer stream.Close()
	var advErr error

	for {
		resp, err := stream.Recv()
		if err != nil {
			if err != io.EOF {
				advErr = err
			}
			break
		}

		// TODO: sort out events and TTL
		advert := &router.Advert{
			Id:        resp.Id,
			Type:      router.AdvertType(resp.Type),
			Timestamp: time.Unix(0, resp.Timestamp),
			//Events:    events,
		}

		select {
		case s.advertChan <- advert:
		case <-s.exit:
			return nil
		}
	}

	return advErr
}

// Advertise advertises routes to the network
func (s *svc) Advertise() (<-chan *router.Advert, error) {
	s.Lock()
	defer s.Unlock()

	switch s.status.Code {
	case router.Advertising:
		return s.advertChan, nil
	case router.Running:
		stream, err := s.router.Advertise(context.Background(), &pb.AdvertiseRequest{})
		if err != nil {
			return nil, fmt.Errorf("failed getting advert stream: %s", err)
		}

		// create advertise and event channels
		s.advertChan = make(chan *router.Advert)

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			select {
			case s.errChan <- s.advertiseEvents(stream):
			case <-s.exit:
			}
		}()

		// mark router as Running and set its Error to nil
		s.status = router.Status{Code: router.Advertising, Error: nil}

		return s.advertChan, nil
	case router.Stopped:
		return nil, fmt.Errorf("not running")
	}

	return nil, fmt.Errorf("error: %s", s.status.Error)
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
	s.Lock()
	// only close the channel if the router is running and/or advertising
	if s.status.Code == router.Running || s.status.Code == router.Advertising {
		// notify all goroutines to finish
		close(s.exit)
		// TODO: might need to drain some channels here

		// mark the router as Stopped and set its Error to nil
		s.status = router.Status{Code: router.Stopped, Error: nil}
	}
	s.Unlock()

	// wait for all goroutines to finish
	s.wg.Wait()

	return nil
}

// Returns the router implementation
func (s *svc) String() string {
	return "service"
}
