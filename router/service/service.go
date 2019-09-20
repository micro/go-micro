package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/router"
	pb "github.com/micro/go-micro/router/proto"
)

type svc struct {
	sync.RWMutex
	opts       router.Options
	callOpts   []client.CallOption
	router     pb.RouterService
	table      *table
	status     *router.Status
	exit       chan bool
	errChan    chan error
	advertChan chan *router.Advert
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
	cli := client.DefaultClient

	// set options client
	if options.Client != nil {
		cli = options.Client
	}

	// set the status to Stopped
	status := &router.Status{
		Code:  router.Stopped,
		Error: nil,
	}

	// NOTE: should we have Client/Service option in router.Options?
	s := &svc{
		opts:   options,
		status: status,
		router: pb.NewRouterService(router.DefaultName, cli),
	}

	// set the router address to call
	if len(options.Address) > 0 {
		s.callOpts = []client.CallOption{
			client.WithAddress(options.Address),
		}
	}
	// set the table
	s.table = &table{pb.NewTableService(router.DefaultName, cli), s.callOpts}

	return s
}

// Init initializes router with given options
func (s *svc) Init(opts ...router.Option) error {
	s.Lock()
	defer s.Unlock()

	for _, o := range opts {
		o(&s.opts)
	}

	return nil
}

// Options returns router options
func (s *svc) Options() router.Options {
	s.Lock()
	opts := s.opts
	s.Unlock()

	return opts
}

// Table returns routing table
func (s *svc) Table() router.Table {
	return s.table
}

// Start starts the service
func (s *svc) Start() error {
	s.Lock()
	defer s.Unlock()

	s.status = &router.Status{
		Code:  router.Running,
		Error: nil,
	}

	return nil
}

func (s *svc) advertiseEvents(advertChan chan *router.Advert, stream pb.Router_AdvertiseService) error {
	go func() {
		<-s.exit
		stream.Close()
	}()

	var advErr error

	for {
		resp, err := stream.Recv()
		if err != nil {
			if err != io.EOF {
				advErr = err
			}
			break
		}

		events := make([]*router.Event, len(resp.Events))
		for i, event := range resp.Events {
			route := router.Route{
				Service: event.Route.Service,
				Address: event.Route.Address,
				Gateway: event.Route.Gateway,
				Network: event.Route.Network,
				Link:    event.Route.Link,
				Metric:  int(event.Route.Metric),
			}

			events[i] = &router.Event{
				Type:      router.EventType(event.Type),
				Timestamp: time.Unix(0, event.Timestamp),
				Route:     route,
			}
		}

		advert := &router.Advert{
			Id:        resp.Id,
			Type:      router.AdvertType(resp.Type),
			Timestamp: time.Unix(0, resp.Timestamp),
			TTL:       time.Duration(resp.Ttl),
			Events:    events,
		}

		select {
		case advertChan <- advert:
		case <-s.exit:
			close(advertChan)
			return nil
		}
	}

	// close the channel on exit
	close(advertChan)

	return advErr
}

// Advertise advertises routes to the network
func (s *svc) Advertise() (<-chan *router.Advert, error) {
	s.Lock()
	defer s.Unlock()

	switch s.status.Code {
	case router.Running, router.Advertising:
		stream, err := s.router.Advertise(context.Background(), &pb.Request{}, s.callOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed getting advert stream: %s", err)
		}
		// create advertise and event channels
		advertChan := make(chan *router.Advert)
		go s.advertiseEvents(advertChan, stream)
		return advertChan, nil
	case router.Stopped:
		return nil, fmt.Errorf("not running")
	}

	return nil, fmt.Errorf("error: %s", s.status.Error)
}

// Process processes incoming adverts
func (s *svc) Process(advert *router.Advert) error {
	var events []*pb.Event
	for _, event := range advert.Events {
		route := &pb.Route{
			Service: event.Route.Service,
			Address: event.Route.Address,
			Gateway: event.Route.Gateway,
			Network: event.Route.Network,
			Link:    event.Route.Link,
			Metric:  int64(event.Route.Metric),
		}
		e := &pb.Event{
			Type:      pb.EventType(event.Type),
			Timestamp: event.Timestamp.UnixNano(),
			Route:     route,
		}
		events = append(events, e)
	}

	advertReq := &pb.Advert{
		Id:        s.Options().Id,
		Type:      pb.AdvertType(advert.Type),
		Timestamp: advert.Timestamp.UnixNano(),
		Events:    events,
	}

	if _, err := s.router.Process(context.Background(), advertReq, s.callOpts...); err != nil {
		return err
	}

	return nil
}

// Solicit advertise all routes
func (s *svc) Solicit() error {
	// list all the routes
	routes, err := s.table.List()
	if err != nil {
		return err
	}

	// build events to advertise
	events := make([]*router.Event, len(routes))
	for i, _ := range events {
		events[i] = &router.Event{
			Type:      router.Update,
			Timestamp: time.Now(),
			Route:     routes[i],
		}
	}

	advert := &router.Advert{
		Id:        s.opts.Id,
		Type:      router.RouteUpdate,
		Timestamp: time.Now(),
		TTL:       time.Duration(router.DefaultAdvertTTL),
		Events:    events,
	}

	select {
	case s.advertChan <- advert:
	case <-s.exit:
		close(s.advertChan)
		return nil
	}

	return nil
}

// Status returns router status
func (s *svc) Status() router.Status {
	s.Lock()
	defer s.Unlock()

	// check if its stopped
	select {
	case <-s.exit:
		return router.Status{
			Code:  router.Stopped,
			Error: nil,
		}
	default:
		// don't block
	}

	// check the remote router
	rsp, err := s.router.Status(context.Background(), &pb.Request{}, s.callOpts...)
	if err != nil {
		return router.Status{
			Code:  router.Error,
			Error: err,
		}
	}

	code := router.Running
	var serr error

	switch rsp.Status.Code {
	case "running":
		code = router.Running
	case "advertising":
		code = router.Advertising
	case "stopped":
		code = router.Stopped
	case "error":
		code = router.Error
	}

	if len(rsp.Status.Error) > 0 {
		serr = errors.New(rsp.Status.Error)
	}

	return router.Status{
		Code:  code,
		Error: serr,
	}
}

// Remote router cannot be stopped
func (s *svc) Stop() error {
	s.Lock()
	defer s.Unlock()

	select {
	case <-s.exit:
		return nil
	default:
		close(s.exit)
	}

	return nil
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
	}, s.callOpts...)

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
	rsp, err := s.router.Watch(context.Background(), &pb.WatchRequest{}, s.callOpts...)
	if err != nil {
		return nil, err
	}
	options := router.WatchOptions{
		Service: "*",
	}
	for _, o := range opts {
		o(&options)
	}
	return newWatcher(rsp, options)
}

// Returns the router implementation
func (s *svc) String() string {
	return "service"
}
