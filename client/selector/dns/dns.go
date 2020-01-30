// Package dns provides a dns SRV selector
package dns

import (
	"fmt"
	"net"
	"strconv"

	"github.com/micro/go-micro/v2/client/selector"
	"github.com/micro/go-micro/v2/registry"
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
	var srv []*net.SRV

	// check if its host:port
	host, port, err := net.SplitHostPort(service)
	// not host:port
	if err != nil {
		// lookup the SRV record
		_, srvs, err := net.LookupSRV(service, "tcp", d.domain)
		if err != nil {
			return nil, err
		}
		// set SRV records
		srv = srvs
		// got host:port
	} else {
		p, _ := strconv.Atoi(port)

		// lookup the A record
		ips, err := net.LookupHost(host)
		if err != nil {
			return nil, err
		}

		// create SRV records
		for _, ip := range ips {
			srv = append(srv, &net.SRV{
				Target: ip,
				Port:   uint16(p),
			})
		}
	}

	nodes := make([]*registry.Node, 0, len(srv))
	for _, node := range srv {
		nodes = append(nodes, &registry.Node{
			Id:      node.Target,
			Address: fmt.Sprintf("%s:%d", node.Target, node.Port),
		})
	}

	services := []*registry.Service{
		{
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

func (d *dnsSelector) Mark(service string, node *registry.Node, err error) {}

func (d *dnsSelector) Reset(service string) {}

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
