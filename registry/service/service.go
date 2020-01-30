// Package service uses the registry service
package service

import (
	"context"
	"time"

	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/client/grpc"
	"github.com/micro/go-micro/v2/registry"
	pb "github.com/micro/go-micro/v2/registry/service/proto"
)

var (
	// The default service name
	DefaultService = "go.micro.registry"
)

type serviceRegistry struct {
	opts registry.Options
	// name of the registry
	name string
	// address
	address []string
	// client to call registry
	client pb.RegistryService
}

func (s *serviceRegistry) callOpts() []client.CallOption {
	var opts []client.CallOption

	// set registry address
	if len(s.address) > 0 {
		opts = append(opts, client.WithAddress(s.address...))
	}

	// set timeout
	if s.opts.Timeout > time.Duration(0) {
		opts = append(opts, client.WithRequestTimeout(s.opts.Timeout))
	}

	return opts
}

func (s *serviceRegistry) Init(opts ...registry.Option) error {
	for _, o := range opts {
		o(&s.opts)
	}
	return nil
}

func (s *serviceRegistry) Options() registry.Options {
	return s.opts
}

func (s *serviceRegistry) Register(srv *registry.Service, opts ...registry.RegisterOption) error {
	var options registry.RegisterOptions
	for _, o := range opts {
		o(&options)
	}

	// encode srv into protobuf and pack Register TTL into it
	pbSrv := ToProto(srv)
	pbSrv.Options.Ttl = int64(options.TTL.Seconds())

	// register the service
	_, err := s.client.Register(context.TODO(), pbSrv, s.callOpts()...)
	if err != nil {
		return err
	}

	return nil
}

func (s *serviceRegistry) Deregister(srv *registry.Service) error {
	// deregister the service
	_, err := s.client.Deregister(context.TODO(), ToProto(srv), s.callOpts()...)
	if err != nil {
		return err
	}
	return nil
}

func (s *serviceRegistry) GetService(name string) ([]*registry.Service, error) {
	rsp, err := s.client.GetService(context.TODO(), &pb.GetRequest{
		Service: name,
	}, s.callOpts()...)

	if err != nil {
		return nil, err
	}

	services := make([]*registry.Service, 0, len(rsp.Services))
	for _, service := range rsp.Services {
		services = append(services, ToService(service))
	}
	return services, nil
}

func (s *serviceRegistry) ListServices() ([]*registry.Service, error) {
	rsp, err := s.client.ListServices(context.TODO(), &pb.ListRequest{}, s.callOpts()...)
	if err != nil {
		return nil, err
	}

	services := make([]*registry.Service, 0, len(rsp.Services))
	for _, service := range rsp.Services {
		services = append(services, ToService(service))
	}

	return services, nil
}

func (s *serviceRegistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	var options registry.WatchOptions
	for _, o := range opts {
		o(&options)
	}

	stream, err := s.client.Watch(context.TODO(), &pb.WatchRequest{
		Service: options.Service,
	}, s.callOpts()...)

	if err != nil {
		return nil, err
	}

	return newWatcher(stream), nil
}

func (s *serviceRegistry) String() string {
	return "service"
}

// NewRegistry returns a new registry service client
func NewRegistry(opts ...registry.Option) registry.Registry {
	var options registry.Options
	for _, o := range opts {
		o(&options)
	}

	// the registry address
	addrs := options.Addrs

	if len(addrs) == 0 {
		addrs = []string{"127.0.0.1:8000"}
	}

	// use mdns as a fall back in case its used
	mReg := registry.NewRegistry()

	// create new client with mdns
	cli := grpc.NewClient(
		client.Registry(mReg),
	)

	// service name
	// TODO: accept option
	name := DefaultService

	return &serviceRegistry{
		opts:    options,
		name:    name,
		address: addrs,
		client:  pb.NewRegistryService(name, cli),
	}
}
