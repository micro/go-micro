package micro

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

type service struct {
	opts Options

	once sync.Once
}

func newService(opts ...Option) Service {
	return &service{
		opts: newOptions(opts...),
	}
}

func (s *service) Name() string {
	return s.opts.Server.Options().Name
}

// Init initializes options. Additionally it calls cmd.Init
// which parses command line flags. cmd.Init is only called
// on first Init.
func (s *service) Init(opts ...Option) {
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

		// If the store has no Table set, fallback to the
		// services name
		if len(s.opts.Store.Options().Table) == 0 {
			name := s.opts.Cmd.App().Name
			err := s.opts.Store.Init(store.Table(name))
			if err != nil {
				s.opts.Logger.Log(log.FatalLevel, err)
			}
		}
	})
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
	return "micro"
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

	for _, fn := range s.opts.AfterStart {
		if err := fn(); err != nil {
			return err
		}
	}

	return nil
}

func (s *service) Stop() error {
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

func (s *service) Run() (err error) {
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
