package roundrobin

import (
	"github.com/micro/go-micro/v3/selector"
)

// NewSelector returns an initalised round robin selector
func NewSelector(opts ...selector.Option) selector.Selector {
	return new(roundrobin)
}

type roundrobin struct{}

func (r *roundrobin) Select(routes []string, opts ...selector.SelectOption) (selector.Next, error) {
	if len(routes) == 0 {
		return nil, selector.ErrNoneAvailable
	}

	var i int

	return func() string {
		route := routes[i%len(routes)]
		// increment
		i++
		return route
	}, nil
}

func (r *roundrobin) Record(addr string, err error) error { return nil }

func (r *roundrobin) Reset() error { return nil }

func (r *roundrobin) String() string {
	return "roundrobin"
}
