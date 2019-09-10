package micro

import (
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/config/cmd"
	"github.com/micro/go-micro/debug/handler"
	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/plugin"
	"github.com/micro/go-micro/server"
	"github.com/micro/go-micro/util/log"
)

type service struct {
	opts Options

	once sync.Once
}

func newService(opts ...Option) Service {
	options := newOptions(opts...)

	options.Client = &clientWrapper{
		options.Client,
		metadata.Metadata{
			HeaderPrefix + "From-Service": options.Server.Options().Name,
		},
	}

	return &service{
		opts: options,
	}
}

// Init initialises options. Additionally it calls cmd.Init
// which parses command line flags. cmd.Init is only called
// on first Init.
func (s *service) Init(opts ...Option) {
	// process options
	for _, o := range opts {
		o(&s.opts)
	}

	s.once.Do(func() {
		// setup the plugins
		for _, p := range strings.Split(os.Getenv("MICRO_PLUGIN"), ",") {
			if len(p) == 0 {
				continue
			}

			// load the plugin
			c, err := plugin.Load(p)
			if err != nil {
				log.Fatal(err)
			}

			// initialise the plugin
			if err := plugin.Init(c); err != nil {
				log.Fatal(err)
			}
		}

		// Initialise the command flags, overriding new service
		_ = s.opts.Cmd.Init(
			cmd.Broker(&s.opts.Broker),
			cmd.Registry(&s.opts.Registry),
			cmd.Transport(&s.opts.Transport),
			cmd.Client(&s.opts.Client),
			cmd.Server(&s.opts.Server),
		)
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

func (s *service) Run() error {
	// register the debug handler
	s.opts.Server.Handle(
		s.opts.Server.NewHandler(
			handler.DefaultHandler,
			server.InternalHandler(true),
		),
	)

	if err := s.Start(); err != nil {
		return err
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	select {
	// wait on kill signal
	case <-ch:
	// wait on context cancel
	case <-s.opts.Context.Done():
	}

	return s.Stop()
}
