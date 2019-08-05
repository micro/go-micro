// Package http provides a http server with features; acme, cors, etc
package http

import (
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/handlers"
	"github.com/micro/go-micro/api/server"
	"github.com/micro/go-micro/util/log"
	"golang.org/x/crypto/acme/autocert"
)

type httpServer struct {
	mux  *http.ServeMux
	opts server.Options

	mtx     sync.RWMutex
	address string
	exit    chan chan error
}

func NewServer(address string) server.Server {
	return &httpServer{
		opts:    server.Options{},
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
	s.mux.Handle(path, handlers.CombinedLoggingHandler(os.Stdout, handler))
}

func (s *httpServer) Start() error {
	var l net.Listener
	var err error

	if s.opts.EnableACME {
		// should we check the address to make sure its using :443?
		l = autocert.NewListener(s.opts.ACMEHosts...)
	} else if s.opts.EnableTLS && s.opts.TLSConfig != nil {
		l, err = tls.Listen("tcp", s.address, s.opts.TLSConfig)
	} else {
		// otherwise plain listen
		l, err = net.Listen("tcp", s.address)
	}
	if err != nil {
		return err
	}

	log.Logf("HTTP API Listening on %s", l.Addr().String())

	s.mtx.Lock()
	s.address = l.Addr().String()
	s.mtx.Unlock()

	go func() {
		if err := http.Serve(l, s.mux); err != nil {
			// temporary fix
			//log.Fatal(err)
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
