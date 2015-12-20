package gomicro

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/context"
	"github.com/micro/go-micro/server"
)

type service struct {
	opts Options
}

func newService(opts ...Option) Service {
	options := newOptions(opts...)

	options.Client = &clientWrap{
		options.Client,
		context.Metadata{
			HeaderPrefix + "From-Service": options.Server.Config().Name(),
		},
	}

	return &service{
		opts: options,
	}
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

	return nil
}

func (s *service) Run() error {
	if err := s.Start(); err != nil {
		return err
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)

	if err := s.Stop(); err != nil {
		return err
	}

	return nil
}
