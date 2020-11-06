// Package rpc provides an app implementation for Nitro
package rpc

import (
	"github.com/asim/nitro/v3/client"
	crpc "github.com/asim/nitro/v3/client/rpc"
	"github.com/asim/nitro/v3/server"
	srpc "github.com/asim/nitro/v3/server/rpc"
	"github.com/asim/nitro/v3/app"
)

type rpcApp struct {
	opts app.Options
}

func newApp(opts ...app.Option) app.App {
	options := app.NewOptions(opts...)

	return &rpcApp{
		opts: options,
	}
}

func (s *rpcApp) Name() string {
	return s.opts.Server.Options().Name
}

// Init initialises options. Additionally it calls cmd.Init
// which parses command line flags. cmd.Init is only called
// on first Init.
func (s *rpcApp) Init(opts ...app.Option) {
	// process options
	for _, o := range opts {
		o(&s.opts)
	}
}

func (s *rpcApp) Options() app.Options {
	return s.opts
}

func (s *rpcApp) Client() client.Client {
	return s.opts.Client
}

func (s *rpcApp) Server() server.Server {
	return s.opts.Server
}

func (s *rpcApp) String() string {
	return "rpc"
}

func (s *rpcApp) Start() error {
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

func (s *rpcApp) Stop() error {
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

func (s *rpcApp) Run() error {
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
		app.Client(crpc.NewClient()),
		app.Server(srpc.NewServer()),
	}

	options = append(options, opts...)

	return newApp(options...)
}
