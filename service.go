package micro

import (
	"os"
	"os/signal"
	"syscall"
	"time"

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

func (s *service) run(exit chan bool) {
	if s.opts.RegisterInterval <= time.Duration(0) {
		return
	}

	t := time.NewTicker(s.opts.RegisterInterval)

	for {
		select {
		case <-t.C:
			s.opts.Server.Register()
		case <-exit:
			t.Stop()
			return
		}
	}
}

func (s *service) Init(opts ...Option) {
	// We might get more command flags or the action here
	// This is pretty ugly, find a better way
	options := newOptions()
	options.Cmd = s.opts.Cmd
	for _, o := range opts {
		o(&options)
	}
	s.opts.Cmd = options.Cmd

	// Initialise the command flags, overriding new service
	s.opts.Cmd.Init(
		cmd.Broker(&s.opts.Broker),
		cmd.Registry(&s.opts.Registry),
		cmd.Transport(&s.opts.Transport),
		cmd.Client(&s.opts.Client),
		cmd.Server(&s.opts.Server),
	)

	// Update any options to override command flags
	for _, o := range opts {
		o(&s.opts)
	}
}

func (s *service) Options() Options {
	return s.opts
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

	// start reg loop
	ex := make(chan bool)
	go s.run(ex)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	<-ch

	// exit reg loop
	close(ex)

	if err := s.Stop(); err != nil {
		return err
	}

	return nil
}
