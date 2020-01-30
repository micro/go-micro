package grpc

import (
	"github.com/micro/go-micro/v2/client"
	gclient "github.com/micro/go-micro/v2/client/grpc"
	"github.com/micro/go-micro/v2/server"
	gserver "github.com/micro/go-micro/v2/server/grpc"
	"github.com/micro/go-micro/v2/service"
)

type grpcService struct {
	opts service.Options
}

func newService(opts ...service.Option) service.Service {
	options := service.NewOptions(opts...)

	return &grpcService{
		opts: options,
	}
}

func (s *grpcService) Name() string {
	return s.opts.Server.Options().Name
}

// Init initialises options. Additionally it calls cmd.Init
// which parses command line flags. cmd.Init is only called
// on first Init.
func (s *grpcService) Init(opts ...service.Option) {
	// process options
	for _, o := range opts {
		o(&s.opts)
	}
}

func (s *grpcService) Options() service.Options {
	return s.opts
}

func (s *grpcService) Client() client.Client {
	return s.opts.Client
}

func (s *grpcService) Server() server.Server {
	return s.opts.Server
}

func (s *grpcService) String() string {
	return "grpc"
}

func (s *grpcService) Start() error {
	for _, fn := range s.opts.BeforeStart {
		if err := fn(); err != nil {
			return err
		}
	}

	if err := s.opts.Server.Start(); err != nil {
		return err
	}

	for _, fn := range s.opts.AfterStart {
		if err := fn(); err != nil {
			return err
		}
	}

	return nil
}

func (s *grpcService) Stop() error {
	var gerr error

	for _, fn := range s.opts.BeforeStop {
		if err := fn(); err != nil {
			gerr = err
		}
	}

	if err := s.opts.Server.Stop(); err != nil {
		return err
	}

	for _, fn := range s.opts.AfterStop {
		if err := fn(); err != nil {
			gerr = err
		}
	}

	return gerr
}

func (s *grpcService) Run() error {
	if err := s.Start(); err != nil {
		return err
	}

	// wait on context cancel
	<-s.opts.Context.Done()

	return s.Stop()
}

// NewService returns a grpc service compatible with go-micro.Service
func NewService(opts ...service.Option) service.Service {
	// our grpc client
	c := gclient.NewClient()
	// our grpc server
	s := gserver.NewServer()

	// create options with priority for our opts
	options := []service.Option{
		service.Client(c),
		service.Server(s),
	}

	// append passed in opts
	options = append(options, opts...)

	// generate and return a service
	return newService(options...)
}
