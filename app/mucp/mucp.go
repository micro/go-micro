// Package mucp provides an app implementation for Nitro
package mucp

import (
	"github.com/asim/nitro/v3/client"
	cmucp "github.com/asim/nitro/v3/client/mucp"
	"github.com/asim/nitro/v3/server"
	smucp "github.com/asim/nitro/v3/server/mucp"
	"github.com/asim/nitro/v3/app"
)

type mucpApp struct {
	opts app.Options
}

func newApp(opts ...app.Option) app.App {
	options := app.NewOptions(opts...)

	return &mucpApp{
		opts: options,
	}
}

func (s *mucpApp) Name() string {
	return s.opts.Server.Options().Name
}

// Init initialises options. Additionally it calls cmd.Init
// which parses command line flags. cmd.Init is only called
// on first Init.
func (s *mucpApp) Init(opts ...app.Option) {
	// process options
	for _, o := range opts {
		o(&s.opts)
	}
}

func (s *mucpApp) Options() app.Options {
	return s.opts
}

func (s *mucpApp) Client() client.Client {
	return s.opts.Client
}

func (s *mucpApp) Server() server.Server {
	return s.opts.Server
}

func (s *mucpApp) String() string {
	return "mucp"
}

func (s *mucpApp) Start() error {
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

func (s *mucpApp) Stop() error {
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

func (s *mucpApp) Run() error {
	if err := s.Start(); err != nil {
		return err
	}

	// wait on context cancel
	<-s.opts.Context.Done()

	return s.Stop()
}

// NewApp returns a new Nitro app
func NewApp(opts ...app.Option) app.App {
	options := []app.Option{
		app.Client(cmucp.NewClient()),
		app.Server(smucp.NewServer()),
	}

	options = append(options, opts...)

	return newApp(options...)
}
