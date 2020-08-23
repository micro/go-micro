// Package static is a static router which returns the service name as the address + port
package static

import (
	"fmt"
	"net"

	"github.com/micro/go-micro/v3/router"
)

var (
	// DefaulPort is the port to append where nothing is set
	DefaultPort = 8080
)

// NewRouter returns an initialized static router
func NewRouter(opts ...router.Option) router.Router {
	options := router.DefaultOptions()
	for _, o := range opts {
		o(&options)
	}
	return &static{options}
}

type static struct {
	options router.Options
}

func (s *static) Init(opts ...router.Option) error {
	for _, o := range opts {
		o(&s.options)
	}
	return nil
}

func (s *static) Options() router.Options {
	return s.options
}

func (s *static) Table() router.Table {
	return nil
}

func (s *static) Lookup(service string, opts ...router.LookupOption) ([]router.Route, error) {
	options := router.NewLookup(opts...)

	_, _, err := net.SplitHostPort(service)
	if err == nil {
		// use the address
		options.Address = service
	} else {
		options.Address = fmt.Sprintf("%s:%d", service, DefaultPort)
	}

	return []router.Route{
		router.Route{
			Service: service,
			Address: options.Address,
			Gateway: options.Gateway,
			Network: options.Network,
			Router:  options.Router,
		},
	}, nil
}

func (s *static) Watch(opts ...router.WatchOption) (router.Watcher, error) {
	return nil, nil
}

func (s *static) Close() error {
	return nil
}

func (s *static) String() string {
	return "static"
}
