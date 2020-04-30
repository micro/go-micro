package service

import (
	"context"
	"sync"

	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/runtime"
	pb "github.com/micro/go-micro/v2/runtime/service/proto"
)

type svc struct {
	sync.RWMutex
	options runtime.Options
	runtime pb.RuntimeService
}

// Init initializes runtime with given options
func (s *svc) Init(opts ...runtime.Option) error {
	s.Lock()
	defer s.Unlock()

	for _, o := range opts {
		o(&s.options)
	}

	return nil
}

// Create registers a service in the runtime
func (s *svc) Create(svc *runtime.Service, opts ...runtime.CreateOption) error {
	var options runtime.CreateOptions
	for _, o := range opts {
		o(&options)
	}
	if options.Context == nil {
		options.Context = context.Background()
	}

	// set the default source from MICRO_RUNTIME_SOURCE
	if len(svc.Source) == 0 {
		svc.Source = s.options.Source
	}

	// runtime service create request
	req := &pb.CreateRequest{
		Service: &pb.Service{
			Name:     svc.Name,
			Version:  svc.Version,
			Source:   svc.Source,
			Metadata: svc.Metadata,
		},
		Options: &pb.CreateOptions{
			Command: options.Command,
			Args:    options.Args,
			Env:     options.Env,
			Type:    options.Type,
			Image:   options.Image,
		},
	}

	if _, err := s.runtime.Create(options.Context, req); err != nil {
		return err
	}

	return nil
}

func (s *svc) Logs(service *runtime.Service, opts ...runtime.LogsOption) (runtime.LogStream, error) {
	var options runtime.LogsOptions
	for _, o := range opts {
		o(&options)
	}

	if options.Context == nil {
		options.Context = context.Background()
	}

	ls, err := s.runtime.Logs(options.Context, &pb.LogsRequest{
		Service: service.Name,
		Stream:  options.Stream,
		Count:   options.Count,
	})
	if err != nil {
		return nil, err
	}
	logStream := &serviceLogStream{
		service: service.Name,
		stream:  make(chan runtime.LogRecord),
		stop:    make(chan bool),
	}

	go func() {
		for {
			select {
			// @todo this never seems to return, investigate
			case <-ls.Context().Done():
				logStream.Stop()
			}
		}
	}()

	go func() {
		for {
			select {
			// @todo this never seems to return, investigate
			case <-ls.Context().Done():
				return
			case _, ok := <-logStream.stream:
				if !ok {
					return
				}
			default:
				record := pb.LogRecord{}
				err := ls.RecvMsg(&record)
				if err != nil {
					logStream.Stop()
					return
				}
				logStream.stream <- runtime.LogRecord{
					Message:  record.GetMessage(),
					Metadata: record.GetMetadata(),
				}
			}
		}
	}()
	return logStream, nil
}

type serviceLogStream struct {
	service string
	stream  chan runtime.LogRecord
	sync.Mutex
	stop chan bool
	err  error
}

func (l *serviceLogStream) Error() error {
	return l.err
}

func (l *serviceLogStream) Chan() chan runtime.LogRecord {
	return l.stream
}

func (l *serviceLogStream) Stop() error {
	l.Lock()
	defer l.Unlock()
	select {
	case <-l.stop:
		return nil
	default:
		close(l.stream)
		close(l.stop)
	}
	return nil
}

// Read returns the service with the given name from the runtime
func (s *svc) Read(opts ...runtime.ReadOption) ([]*runtime.Service, error) {
	var options runtime.ReadOptions
	for _, o := range opts {
		o(&options)
	}
	if options.Context == nil {
		options.Context = context.Background()
	}

	// runtime service create request
	req := &pb.ReadRequest{
		Options: &pb.ReadOptions{
			Service: options.Service,
			Version: options.Version,
			Type:    options.Type,
		},
	}

	resp, err := s.runtime.Read(options.Context, req)
	if err != nil {
		return nil, err
	}

	services := make([]*runtime.Service, 0, len(resp.Services))
	for _, service := range resp.Services {
		svc := &runtime.Service{
			Name:     service.Name,
			Version:  service.Version,
			Source:   service.Source,
			Metadata: service.Metadata,
		}
		services = append(services, svc)
	}

	return services, nil
}

// Update updates the running service
func (s *svc) Update(svc *runtime.Service, opts ...runtime.UpdateOption) error {
	var options runtime.UpdateOptions
	for _, o := range opts {
		o(&options)
	}
	if options.Context == nil {
		options.Context = context.Background()
	}

	// runtime service create request
	req := &pb.UpdateRequest{
		Service: &pb.Service{
			Name:     svc.Name,
			Version:  svc.Version,
			Source:   svc.Source,
			Metadata: svc.Metadata,
		},
	}

	if _, err := s.runtime.Update(options.Context, req); err != nil {
		return err
	}

	return nil
}

// Delete stops and removes the service from the runtime
func (s *svc) Delete(svc *runtime.Service, opts ...runtime.DeleteOption) error {
	var options runtime.DeleteOptions
	for _, o := range opts {
		o(&options)
	}
	if options.Context == nil {
		options.Context = context.Background()
	}

	// runtime service create request
	req := &pb.DeleteRequest{
		Service: &pb.Service{
			Name:     svc.Name,
			Version:  svc.Version,
			Source:   svc.Source,
			Metadata: svc.Metadata,
		},
	}

	if _, err := s.runtime.Delete(options.Context, req); err != nil {
		return err
	}

	return nil
}

// Start starts the runtime
func (s *svc) Start() error {
	// NOTE: nothing to be done here
	return nil
}

// Stop stops the runtime
func (s *svc) Stop() error {
	// NOTE: nothing to be done here
	return nil
}

// Returns the runtime service implementation
func (s *svc) String() string {
	return "service"
}

// NewRuntime creates new service runtime and returns it
func NewRuntime(opts ...runtime.Option) runtime.Runtime {
	// get default options
	options := runtime.Options{}

	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	// create default client
	cli := client.DefaultClient

	return &svc{
		options: options,
		runtime: pb.NewRuntimeService(runtime.DefaultName, cli),
	}
}
