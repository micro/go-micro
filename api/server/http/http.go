// Package http provides a http server with features; acme, cors, etc
package http

import (
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/handlers"
	"github.com/micro/go-micro/v2/api/server"
	"github.com/micro/go-micro/v2/api/server/cors"
	"github.com/micro/go-micro/v2/logger"
)

type httpServer struct {
	mux  *http.ServeMux
	opts server.Options

	mtx     sync.RWMutex
	address string
	exit    chan chan error
}

func NewServer(address string, opts ...server.Option) server.Server {
	var options server.Options
	for _, o := range opts {
		o(&options)
	}

	return &httpServer{
		opts:    options,
		mux:     http.NewServeMux(),
		address: address,
		exit:    make(chan chan error),
	}
}

func (s *httpServer) Address() string {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.address
}

func (s *httpServer) Init(opts ...server.Option) error {
	for _, o := range opts {
		o(&s.opts)
	}
	return nil
}

func (s *httpServer) Handle(path string, handler http.Handler) {
	// TODO: move this stuff out to one place with ServeHTTP

	// apply the wrappers, e.g. auth
	for _, wrapper := range s.opts.Wrappers {
		handler = wrapper(handler)
	}

	// wrap with cors
	if s.opts.EnableCORS {
		handler = cors.CombinedCORSHandler(handler)
	}

	// wrap with logger
	handler = handlers.CombinedLoggingHandler(os.Stdout, handler)

	s.mux.Handle(path, handler)
}

func (s *httpServer) Start() error {
	var l net.Listener
	var err error

	if s.opts.EnableACME && s.opts.ACMEProvider != nil {
		// should we check the address to make sure its using :443?
		l, err = s.opts.ACMEProvider.Listen(s.opts.ACMEHosts...)
	} else if s.opts.EnableTLS && s.opts.TLSConfig != nil {
		l, err = tls.Listen("tcp", s.address, s.opts.TLSConfig)
	} else {
		// otherwise plain listen
		l, err = net.Listen("tcp", s.address)
	}
	if err != nil {
		return err
	}

	if logger.V(logger.InfoLevel, logger.DefaultLogger) {
		logger.Infof("HTTP API Listening on %s", l.Addr().String())
	}

	s.mtx.Lock()
	s.address = l.Addr().String()
	s.mtx.Unlock()

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
