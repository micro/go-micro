// Package static provides a static resolver which returns the name/ip passed in without any change
package static

import (
	"net"
	"strconv"

	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/selector"
)

// staticSelector is a static selector
type staticSelector struct {
	opts selector.Options
}

func (s *staticSelector) Init(opts ...selector.Option) error {
	for _, o := range opts {
		o(&s.opts)
	}
	return nil
}

func (s *staticSelector) Options() selector.Options {
	return s.opts
}

func (s *staticSelector) Select(service string, opts ...selector.SelectOption) (selector.Next, error) {
	var port int
	addr, pt, err := net.SplitHostPort(service)
	if err != nil {
		addr = service
		port = 0
	} else {
		port, _ = strconv.Atoi(pt)
	}

	return func() (*registry.Node, error) {
		return &registry.Node{
			Id:      service,
			Address: addr,
			Port:    port,
		}, nil
	}, nil
}

func (s *staticSelector) Mark(service string, node *registry.Node, err error) {
	return
}

func (s *staticSelector) Reset(service string) {
	return
}

func (s *staticSelector) Close() error {
	return nil
}

func (s *staticSelector) String() string {
	return "static"
}

func NewSelector(opts ...selector.Option) selector.Selector {
	var options selector.Options
	for _, o := range opts {
		o(&options)
	}
	return &staticSelector{
		opts: options,
	}
}
