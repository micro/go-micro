package service

import (
	"context"
	"sync"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/runtime"
	pb "github.com/micro/go-micro/runtime/service/proto"
)

type svc struct {
	sync.RWMutex
	options runtime.Options
	runtime pb.RuntimeService
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
	options := runtime.CreateOptions{}
	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	// runtime service create request
	req := &pb.CreateRequest{
		Service: &pb.Service{
			Name:    svc.Name,
			Version: svc.Version,
			Source:  svc.Source,
		},
		Options: &pb.CreateOptions{
			Command: options.Command,
			Env:     options.Env,
		},
	}

	if _, err := s.runtime.Create(context.Background(), req); err != nil {
		return err
	}

	return nil
}

// Get returns the service with the given name from the runtime
func (s *svc) Get(name string, opts ...runtime.GetOption) ([]*runtime.Service, error) {
	options := runtime.GetOptions{}
	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	// runtime service create request
	req := &pb.GetRequest{
		Name: name,
		Options: &pb.GetOptions{
			Version: options.Version,
		},
	}

	resp, err := s.runtime.Get(context.Background(), req)
	if err != nil {
		return nil, err
	}

	services := make([]*runtime.Service, 0, len(resp.Services))
	for _, service := range resp.Services {
		svc := &runtime.Service{
			Name:    service.Name,
			Version: service.Version,
			Source:  service.Source,
			Path:    service.Path,
			Exec:    service.Exec,
		}
		services = append(services, svc)
	}

	return services, nil
}

// Update updates the running service
func (s *svc) Update(svc *runtime.Service) error {
	// runtime service create request
	req := &pb.UpdateRequest{
		Service: &pb.Service{
			Name:    svc.Name,
			Version: svc.Version,
		},
	}

	if _, err := s.runtime.Update(context.Background(), req); err != nil {
		return err
	}

	return nil
}

// Delete stops and removes the service from the runtime
func (s *svc) Delete(svc *runtime.Service) error {
	// runtime service create request
	req := &pb.DeleteRequest{
		Service: &pb.Service{
			Name:    svc.Name,
			Version: svc.Version,
		},
	}

	if _, err := s.runtime.Delete(context.Background(), req); err != nil {
		return err
	}

	return nil
}

// List lists all services managed by the runtime
func (s *svc) List() ([]*runtime.Service, error) {
	// list all services managed by the runtime
	resp, err := s.runtime.List(context.Background(), &pb.ListRequest{})
	if err != nil {
		return nil, err
	}

	services := make([]*runtime.Service, 0, len(resp.Services))
	for _, service := range resp.Services {
		svc := &runtime.Service{
			Name:    service.Name,
			Version: service.Version,
			Source:  service.Source,
			Path:    service.Path,
			Exec:    service.Exec,
		}
		services = append(services, svc)
	}

	return services, nil
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
