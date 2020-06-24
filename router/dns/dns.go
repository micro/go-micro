package dns

import (
	"fmt"
	"net"
	"strconv"

	"github.com/micro/go-micro/v2/router"
)

// NewRouter returns an initialized dns router
func NewRouter(opts ...router.Option) router.Router {
	options := router.DefaultOptions()
	for _, o := range opts {
		o(&options)
	}
	if len(options.Network) == 0 {
		options.Network = "micro"
	}
	return &dns{options, &table{options}}
}

type dns struct {
	options router.Options
	table   *table
}

func (d *dns) Init(opts ...router.Option) error {
	for _, o := range opts {
		o(&d.options)
	}
	d.table.options = d.options
	return nil
}

func (d *dns) Options() router.Options {
	return d.options
}

func (d *dns) Table() router.Table {
	return d.table
}

func (d *dns) Advertise() (<-chan *router.Advert, error) {
	return nil, nil
}

func (d *dns) Process(*router.Advert) error {
	return nil
}

func (d *dns) Lookup(opts ...router.QueryOption) ([]router.Route, error) {
	return d.table.Query(opts...)
}

func (d *dns) Watch(opts ...router.WatchOption) (router.Watcher, error) {
	return nil, nil
}

func (d *dns) Close() error {
	return nil
}

func (d *dns) String() string {
	return "dns"
}

type table struct {
	options router.Options
}

func (t *table) Create(router.Route) error {
	return nil
}

func (t *table) Delete(router.Route) error {
	return nil
}

func (t *table) Update(router.Route) error {
	return nil
}

func (t *table) List() ([]router.Route, error) {
	return nil, nil
}

func (t *table) Query(opts ...router.QueryOption) ([]router.Route, error) {
	options := router.NewQuery(opts...)

	// check to see if we have the port provided in the service, e.g. go-micro-srv-foo:8000
	host, port, err := net.SplitHostPort(options.Service)
	if err == nil {
		// lookup the service using A records
		ips, err := net.LookupHost(host)
		if err != nil {
			return nil, err
		}

		p, _ := strconv.Atoi(port)

		// convert the ip addresses to routes
		result := make([]router.Route, len(ips))
		for i, ip := range ips {
			result[i] = router.Route{
				Service: options.Service,
				Address: fmt.Sprintf("%s:%d", ip, uint16(p)),
			}
		}
		return result, nil
	}

	// we didn't get the port so we'll lookup the service using SRV records. If we can't lookup the
	// service using the SRV record, we return the error.
	_, nodes, err := net.LookupSRV(options.Service, "tcp", t.options.Network)
	if err != nil {
		return nil, err
	}

	// convert the nodes (net services) to routes
	result := make([]router.Route, len(nodes))
	for i, n := range nodes {
		result[i] = router.Route{
			Service: options.Service,
			Address: fmt.Sprintf("%s:%d", n.Target, n.Port),
			Network: t.options.Network,
		}
	}
	return result, nil
}
