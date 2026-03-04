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

// Service is the interface for a go-micro service.
type Service interface {
	// Name returns the service name.
	Name() string
	// Init initializes options. Parses command line flags on first call.
	Init(...Option)
	// Options returns the current options.
	Options() Options
	// Handle registers a handler with optional server.HandlerOption args.
	Handle(v interface{}, opts ...server.HandlerOption) error
	// Client returns the RPC client.
	Client() client.Client
	// Server returns the RPC server.
	Server() server.Server
	// Start the service (non-blocking).
	Start() error
	// Stop the service.
	Stop() error
	// Run starts the service, blocks on signal/context, then stops.
	Run() error
	// String returns the implementation name.
	String() string
}

type serviceImpl struct {
	opts Options

	once sync.Once
}

// New creates a new service with the given options.
func New(opts ...Option) Service {
	return &serviceImpl{
		opts: newOptions(opts...),
	}
}

func (s *serviceImpl) Name() string {
	return s.opts.Server.Options().Name
}

// Init initializes options. Additionally it calls cmd.Init
// which parses command line flags. cmd.Init is only called
// on first Init.
func (s *serviceImpl) Init(opts ...Option) {
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

		// Initialize the store with the service name as table
		name := s.opts.Cmd.App().Name
		if err := s.opts.Store.Init(store.Table(name)); err != nil {
			s.opts.Logger.Logf(log.ErrorLevel, "error initializing store: %v", err)
		}
	})
}

func (s *serviceImpl) Options() Options {
	return s.opts
}

func (s *serviceImpl) Client() client.Client {
	return s.opts.Client
}

func (s *serviceImpl) Server() server.Server {
	return s.opts.Server
}

func (s *serviceImpl) String() string {
	return "micro"
}

func (s *serviceImpl) Start() error {
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

func (s *serviceImpl) Stop() error {
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

func (s *serviceImpl) Handle(v interface{}, opts ...server.HandlerOption) error {
	return s.opts.Server.Handle(
		s.opts.Server.NewHandler(v, opts...),
	)
}

func (s *serviceImpl) Run() (err error) {
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
