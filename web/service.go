package web

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/micro/cli"
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/registry"
	maddr "github.com/micro/go-micro/util/addr"
	mhttp "github.com/micro/go-micro/util/http"
	"github.com/micro/go-micro/util/log"
	mnet "github.com/micro/go-micro/util/net"
	mls "github.com/micro/go-micro/util/tls"
)

type service struct {
	opts Options

	mux *http.ServeMux
	srv *registry.Service

	sync.Mutex
	running bool
	static  bool
	exit    chan chan error
}

func newService(opts ...Option) Service {
	options := newOptions(opts...)
	s := &service{
		opts:   options,
		mux:    http.NewServeMux(),
		static: true,
	}
	s.srv = s.genSrv()
	return s
}

func (s *service) genSrv() *registry.Service {
	// default host:port
	parts := strings.Split(s.opts.Address, ":")
	host := strings.Join(parts[:len(parts)-1], ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])

	// check the advertise address first
	// if it exists then use it, otherwise
	// use the address
	if len(s.opts.Advertise) > 0 {
		parts = strings.Split(s.opts.Advertise, ":")

		// we have host:port
		if len(parts) > 1 {
			// set the host
			host = strings.Join(parts[:len(parts)-1], ":")

			// get the port
			if aport, _ := strconv.Atoi(parts[len(parts)-1]); aport > 0 {
				port = aport
			}
		} else {
			host = parts[0]
		}
	}

	addr, err := maddr.Extract(host)
	if err != nil {
		// best effort localhost
		addr = "127.0.0.1"
	}

	return &registry.Service{
		Name:    s.opts.Name,
		Version: s.opts.Version,
		Nodes: []*registry.Node{&registry.Node{
			Id:       s.opts.Id,
			Address:  fmt.Sprintf("%s:%d", addr, port),
			Metadata: s.opts.Metadata,
		}},
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
			s.register()
		case <-exit:
			t.Stop()
			return
		}
	}
}

func (s *service) register() error {
	if s.srv == nil {
		return nil
	}
	// default to service registry
	r := s.opts.Service.Client().Options().Registry
	// switch to option if specified
	if s.opts.Registry != nil {
		r = s.opts.Registry
	}
	return r.Register(s.srv, registry.RegisterTTL(s.opts.RegisterTTL))
}

func (s *service) deregister() error {
	if s.srv == nil {
		return nil
	}
	// default to service registry
	r := s.opts.Service.Client().Options().Registry
	// switch to option if specified
	if s.opts.Registry != nil {
		r = s.opts.Registry
	}
	return r.Deregister(s.srv)
}

func (s *service) start() error {
	s.Lock()
	defer s.Unlock()

	if s.running {
		return nil
	}

	l, err := s.listen("tcp", s.opts.Address)
	if err != nil {
		return err
	}

	s.opts.Address = l.Addr().String()
	srv := s.genSrv()
	srv.Endpoints = s.srv.Endpoints
	s.srv = srv

	var h http.Handler

	if s.opts.Handler != nil {
		h = s.opts.Handler
	} else {
		h = s.mux
		var r sync.Once

		// register the html dir
		r.Do(func() {
			// static dir
			static := s.opts.StaticDir
			if s.opts.StaticDir[0] != '/' {
				dir, _ := os.Getwd()
				static = filepath.Join(dir, static)
			}

			// set static if no / handler is registered
			if s.static {
				_, err := os.Stat(static)
				if err == nil {
					log.Logf("Enabling static file serving from %s", static)
					s.mux.Handle("/", http.FileServer(http.Dir(static)))
				}
			}
		})
	}

	for _, fn := range s.opts.BeforeStart {
		if err := fn(); err != nil {
			return err
		}
	}

	var httpSrv *http.Server
	if s.opts.Server != nil {
		httpSrv = s.opts.Server
	} else {
		httpSrv = &http.Server{}
	}

	httpSrv.Handler = h

	go httpSrv.Serve(l)

	for _, fn := range s.opts.AfterStart {
		if err := fn(); err != nil {
			return err
		}
	}

	s.exit = make(chan chan error, 1)
	s.running = true

	go func() {
		ch := <-s.exit
		ch <- l.Close()
	}()

	log.Logf("Listening on %v\n", l.Addr().String())
	return nil
}

func (s *service) stop() error {
	s.Lock()
	defer s.Unlock()

	if !s.running {
		return nil
	}

	for _, fn := range s.opts.BeforeStop {
		if err := fn(); err != nil {
			return err
		}
	}

	ch := make(chan error, 1)
	s.exit <- ch
	s.running = false

	log.Log("Stopping")

	for _, fn := range s.opts.AfterStop {
		if err := fn(); err != nil {
			if chErr := <-ch; chErr != nil {
				return chErr
			}
			return err
		}
	}

	return <-ch
}

func (s *service) Client() *http.Client {
	rt := mhttp.NewRoundTripper(
		mhttp.WithRegistry(registry.DefaultRegistry),
	)
	return &http.Client{
		Transport: rt,
	}
}

func (s *service) Handle(pattern string, handler http.Handler) {
	var seen bool
	for _, ep := range s.srv.Endpoints {
		if ep.Name == pattern {
			seen = true
			break
		}
	}

	// if its unseen then add an endpoint
	if !seen {
		s.srv.Endpoints = append(s.srv.Endpoints, &registry.Endpoint{
			Name: pattern,
		})
	}

	// disable static serving
	if pattern == "/" {
		s.Lock()
		s.static = false
		s.Unlock()
	}

	// register the handler
	s.mux.Handle(pattern, handler)
}

func (s *service) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	var seen bool
	for _, ep := range s.srv.Endpoints {
		if ep.Name == pattern {
			seen = true
			break
		}
	}
	if !seen {
		s.srv.Endpoints = append(s.srv.Endpoints, &registry.Endpoint{
			Name: pattern,
		})
	}

	s.mux.HandleFunc(pattern, handler)
}

func (s *service) Init(opts ...Option) error {
	for _, o := range opts {
		o(&s.opts)
	}

	serviceOpts := []micro.Option{}

	if len(s.opts.Flags) > 0 {
		serviceOpts = append(serviceOpts, micro.Flags(s.opts.Flags...))
	}

	if s.opts.Registry != nil {
		serviceOpts = append(serviceOpts, micro.Registry(s.opts.Registry))
	}

	serviceOpts = append(serviceOpts, micro.Action(func(ctx *cli.Context) {
		if ttl := ctx.Int("register_ttl"); ttl > 0 {
			s.opts.RegisterTTL = time.Duration(ttl) * time.Second
		}

		if interval := ctx.Int("register_interval"); interval > 0 {
			s.opts.RegisterInterval = time.Duration(interval) * time.Second
		}

		if name := ctx.String("server_name"); len(name) > 0 {
			s.opts.Name = name
		}

		if ver := ctx.String("server_version"); len(ver) > 0 {
			s.opts.Version = ver
		}

		if id := ctx.String("server_id"); len(id) > 0 {
			s.opts.Id = id
		}

		if addr := ctx.String("server_address"); len(addr) > 0 {
			s.opts.Address = addr
		}

		if adv := ctx.String("server_advertise"); len(adv) > 0 {
			s.opts.Advertise = adv
		}

		if s.opts.Action != nil {
			s.opts.Action(ctx)
		}
	}))

	s.opts.Service.Init(serviceOpts...)
	srv := s.genSrv()
	srv.Endpoints = s.srv.Endpoints
	s.srv = srv

	return nil
}

func (s *service) Run() error {
	if err := s.start(); err != nil {
		return err
	}

	if err := s.register(); err != nil {
		return err
	}

	// start reg loop
	ex := make(chan bool)
	go s.run(ex)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)

	select {
	// wait on kill signal
	case sig := <-ch:
		log.Logf("Received signal %s\n", sig)
	// wait on context cancel
	case <-s.opts.Context.Done():
		log.Logf("Received context shutdown")
	}

	// exit reg loop
	close(ex)

	if err := s.deregister(); err != nil {
		return err
	}

	return s.stop()
}

// Options returns the options for the given service
func (s *service) Options() Options {
	return s.opts
}

func (s *service) listen(network, addr string) (net.Listener, error) {
	var l net.Listener
	var err error

	// TODO: support use of listen options
	if s.opts.Secure || s.opts.TLSConfig != nil {
		config := s.opts.TLSConfig

		fn := func(addr string) (net.Listener, error) {
			if config == nil {
				hosts := []string{addr}

				// check if its a valid host:port
				if host, _, err := net.SplitHostPort(addr); err == nil {
					if len(host) == 0 {
						hosts = maddr.IPs()
					} else {
						hosts = []string{host}
					}
				}

				// generate a certificate
				cert, err := mls.Certificate(hosts...)
				if err != nil {
					return nil, err
				}
				config = &tls.Config{Certificates: []tls.Certificate{cert}}
			}
			return tls.Listen(network, addr, config)
		}

		l, err = mnet.Listen(addr, fn)
	} else {
		fn := func(addr string) (net.Listener, error) {
			return net.Listen(network, addr)
		}

		l, err = mnet.Listen(addr, fn)
	}

	if err != nil {
		return nil, err
	}

	return l, nil
}
