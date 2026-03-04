package service

import (
	"os"
	"os/signal"
	rtime "runtime"
	"sync"

	"go-micro.dev/v5/client"
	"go-micro.dev/v5/cmd"
	log "go-micro.dev/v5/logger"
	"go-micro.dev/v5/server"
	"go-micro.dev/v5/store"
	signalutil "go-micro.dev/v5/util/signal"
)

// ServiceImpl is the concrete service implementation. It is exported
// to allow the micro package to construct Groups, but users should
// generally interact through the Service interface.
type ServiceImpl struct {
	opts Options

	once sync.Once
}

func New(opts ...Option) *ServiceImpl {
	return &ServiceImpl{
		opts: newOptions(opts...),
	}
}

func (s *ServiceImpl) Name() string {
	return s.opts.Server.Options().Name
}

// Init initializes options. Additionally it calls cmd.Init
// which parses command line flags. cmd.Init is only called
// on first Init.
func (s *ServiceImpl) Init(opts ...Option) {
	// process options
	for _, o := range opts {
		o(&s.opts)
	}

	s.once.Do(func() {
		// set cmd name
		if len(s.opts.Cmd.App().Name) == 0 {
			s.opts.Cmd.App().Name = s.Server().Options().Name
		}

		// Initialize the command flags, overriding new service
		if err := s.opts.Cmd.Init(
			cmd.Auth(&s.opts.Auth),
			cmd.Broker(&s.opts.Broker),
			cmd.Registry(&s.opts.Registry),
			cmd.Transport(&s.opts.Transport),
			cmd.Client(&s.opts.Client),
			cmd.Config(&s.opts.Config),
			cmd.Server(&s.opts.Server),
			cmd.Store(&s.opts.Store),
			cmd.Profile(&s.opts.Profile),
		); err != nil {
			s.opts.Logger.Log(log.FatalLevel, err)
		}

		// we might not want to do this
		name := s.opts.Cmd.App().Name
		err := s.opts.Store.Init(store.Table(name))
		if err != nil {
			s.opts.Logger.Log(log.FatalLevel, err)
		}
	})
}

func (s *ServiceImpl) Options() Options {
	return s.opts
}

func (s *ServiceImpl) Client() client.Client {
	return s.opts.Client
}

func (s *ServiceImpl) Server() server.Server {
	return s.opts.Server
}

func (s *ServiceImpl) String() string {
	return "micro"
}

func (s *ServiceImpl) Start() error {
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

func (s *ServiceImpl) Stop() error {
	var err error

	for _, fn := range s.opts.BeforeStop {
		err = fn()
	}

	if err := s.opts.Server.Stop(); err != nil {
		return err
	}

	for _, fn := range s.opts.AfterStop {
		err = fn()
	}

	return err
}

func (s *ServiceImpl) Handle(v interface{}) error {
	return s.opts.Server.Handle(
		s.opts.Server.NewHandler(v),
	)
}

func (s *ServiceImpl) Run() (err error) {
	logger := s.opts.Logger

	// exit when help flag is provided
	for _, v := range os.Args[1:] {
		if v == "-h" || v == "--help" {
			os.Exit(0)
		}
	}

	// start the profiler
	if s.opts.Profile != nil {
		// to view mutex contention
		rtime.SetMutexProfileFraction(5)
		// to view blocking profile
		rtime.SetBlockProfileRate(1)

		if err = s.opts.Profile.Start(); err != nil {
			return err
		}

		defer func() {
			if nerr := s.opts.Profile.Stop(); nerr != nil {
				logger.Log(log.ErrorLevel, nerr)
			}
		}()
	}

	logger.Logf(log.InfoLevel, "Starting [service] %s", s.Name())

	if err = s.Start(); err != nil {
		return err
	}

	ch := make(chan os.Signal, 1)
	if s.opts.Signal {
		signal.Notify(ch, signalutil.Shutdown()...)
	}

	select {
	// wait on kill signal
	case <-ch:
	// wait on context cancel
	case <-s.opts.Context.Done():
	}

	return s.Stop()
}
