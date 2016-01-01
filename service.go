package gomicro

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/cmd"
	"github.com/micro/go-micro/context"
	"github.com/micro/go-micro/server"
)

type service struct {
	opts Options
}

func newService(opts ...Option) Service {
	options := newOptions(opts...)

	options.Client = &clientWrapper{
		options.Client,
		context.Metadata{
			HeaderPrefix + "From-Service": options.Server.Options().Name,
		},
	}

	return &service{
		opts: options,
	}
}

func (s *service) Init(opts ...Option) {
	s.opts.Cmd.Init()

	for _, o := range opts {
		o(&s.opts)
	}

	s = newService(opts...).(*service)
}

func (s *service) Cmd() cmd.Cmd {
	return s.opts.Cmd
}

func (s *service) Client() client.Client {
	return s.opts.Client
}

func (s *service) Server() server.Server {
	return s.opts.Server
}

func (s *service) String() string {
	return "go-micro"
}

func (s *service) Start() error {
	for _, fn := range s.opts.BeforeStart {
		if err := fn(); err != nil {
			return err
		}
	}

	if err := s.opts.Server.Start(); err != nil {
		return err
	}

	if err := s.opts.Server.Register(); err != nil {
		return err
	}

	return nil
}

func (s *service) Stop() error {
	if err := s.opts.Server.Deregister(); err != nil {
		return err
	}

	if err := s.opts.Server.Stop(); err != nil {
		return err
	}

	var gerr error
	for _, fn := range s.opts.AfterStop {
		if err := fn(); err != nil {
			// should we bail if it fails?
			// other funcs will not be executed
			// seems wrong
			gerr = err
		}
	}
	return gerr
}

func (s *service) Run() error {
	if err := s.Start(); err != nil {
		return err
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	<-ch

	if err := s.Stop(); err != nil {
		return err
	}

	return nil
}
