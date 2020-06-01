package roundrobin

import (
	"github.com/micro/go-micro/v2/router"
	"github.com/micro/go-micro/v2/selector"
)

// NewSelector returns an initalised round robin selector
func NewSelector(opts ...selector.Option) selector.Selector {
	return new(roundrobin)
}

type roundrobin struct{}

// TODO: Implement round robin selector

func (r *roundrobin) Init(opts ...selector.Option) error {
	return nil
}

func (r *roundrobin) Options() selector.Options {
	return selector.Options{}
}

func (r *roundrobin) Select(srvs ...router.Route) (*router.Route, error) {
	return nil, nil
}

func (r *roundrobin) Record(srv *router.Route, err error) error {
	return nil
}

func (r *roundrobin) String() string {
	return "roundrobin"
}
