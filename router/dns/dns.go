package dns

import "github.com/micro/go-micro/v2/router"

// NewRouter returns an initialized dns router
func NewRouter(opts ...router.Option) router.Router {
	var options router.Options
	for _, o := range opts {
		o(&options)
	}

	return &dns{options}
}

type dns struct {
	options router.Options
}

func (d *dns) Init(opts ...router.Option) error {
	for _, o := range opts {
		o(&d.options)
	}
	return nil
}

func (d *dns) Options() router.Options {
	return d.options
}

func (d *dns) Table() router.Table {
	return nil
}

func (d *dns) Advertise() (<-chan *router.Advert, error) {
	return nil, nil
}

func (d *dns) Process(*router.Advert) error {
	return nil
}

func (d *dns) Lookup(...router.QueryOption) ([]router.Route, error) {
	return nil, nil
}

func (d *dns) Watch(opts ...router.WatchOption) (router.Watcher, error) {
	return nil, nil
}

func (d *dns) Start() error {
	return nil
}

func (d *dns) Stop() error {
	return nil
}

func (d *dns) String() string {
	return "dns"
}
