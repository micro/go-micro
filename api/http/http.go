// Package http provides a http server with features; acme, cors, etc
package http

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync"

	"github.com/asim/go-micro/v3/api"
	"github.com/asim/go-micro/v3/logger"
)

type httpServer struct {
	mux  *http.ServeMux
	opts api.Options

	mtx     sync.RWMutex
	address string
	exit    chan chan error
}

// NewGateway returns a new HTTP api gateway
func NewGateway(opts ...api.Option) api.Gateway {
	var options api.Options
	for _, o := range opts {
		o(&options)
	}

	return &httpServer{
		opts: options,
		mux:  http.NewServeMux(),
		exit: make(chan chan error),
	}
}

func (s *httpServer) Init(opts ...api.Option) error {
	for _, o := range opts {
		o(&s.opts)
	}
	return nil
}

func (s *httpServer) Options() api.Options {
	return s.opts
}

func (s *httpServer) Register(ep *api.Endpoint) error   { return nil }
func (s *httpServer) Deregister(ep *api.Endpoint) error { return nil }

func (s *httpServer) Handle(path string, handler http.Handler) {
	s.mux.Handle(path, handler)
}

func (s *httpServer) Serve() error {
	if err := s.Start(); err != nil {
		return err
	}

	<-s.exit
	return nil
}

func (s *httpServer) Start() error {
	var l net.Listener
	var err error

	if s.opts.EnableACME && s.opts.ACMEProvider != nil {
		// should we check the address to make sure its using :443?
		l, err = s.opts.ACMEProvider.Listen(s.opts.ACMEHosts...)
	} else if s.opts.EnableTLS && s.opts.TLSConfig != nil {
		l, err = tls.Listen("tcp", s.opts.Address, s.opts.TLSConfig)
	} else {
		// otherwise plain listen
		l, err = net.Listen("tcp", s.opts.Address)
	}
	if err != nil {
		return err
	}

	if logger.V(logger.InfoLevel, logger.DefaultLogger) {
		logger.Infof("HTTP API Listening on %s", l.Addr().String())
	}

	go func() {
		if err := http.Serve(l, s.mux); err != nil {
			// temporary fix
			//logger.Fatal(err)
		}
	}()

	go func() {
		ch := <-s.exit
		ch <- l.Close()
	}()

	return nil
}

func (s *httpServer) Stop() error {
	ch := make(chan error)
	s.exit <- ch
	return <-ch
}

func (s *httpServer) String() string {
	return "http"
}
