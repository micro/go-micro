// Package dns provides a dns SRV selector
package dns

import (
	"net"

	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/selector"
)

type dnsSelector struct {
	options selector.Options
	domain  string
}

var (
	DefaultDomain = "local"
)

func (d *dnsSelector) Init(opts ...selector.Option) error {
	for _, o := range opts {
		o(&d.options)
	}
	return nil
}

func (d *dnsSelector) Options() selector.Options {
	return d.options
}

func (d *dnsSelector) Select(service string, opts ...selector.SelectOption) (selector.Next, error) {
	_, srv, err := net.LookupSRV(service, "tcp", d.domain)
	if err != nil {
		return nil, err
	}

	var nodes []*registry.Node
	for _, node := range srv {
		nodes = append(nodes, &registry.Node{
			Id:      node.Target,
			Address: node.Target,
			Port:    int(node.Port),
		})
	}

	services := []*registry.Service{
		&registry.Service{
			Name:  service,
			Nodes: nodes,
		},
	}

	sopts := selector.SelectOptions{
		Strategy: d.options.Strategy,
	}

	for _, opt := range opts {
		opt(&sopts)
	}

	// apply the filters
	for _, filter := range sopts.Filters {
		services = filter(services)
	}

	// if there's nothing left, return
	if len(services) == 0 {
		return nil, selector.ErrNoneAvailable
	}

	return sopts.Strategy(services), nil
}

func (d *dnsSelector) Mark(service string, node *registry.Node, err error) {
	return
}

func (d *dnsSelector) Reset(service string) {
	return
}

func (d *dnsSelector) Close() error {
	return nil
}

func (d *dnsSelector) String() string {
	return "dns"
}

func NewSelector(opts ...selector.Option) selector.Selector {
	options := selector.Options{
		Strategy: selector.Random,
	}

	for _, o := range opts {
		o(&options)
	}

	return &dnsSelector{options: options, domain: DefaultDomain}
}
