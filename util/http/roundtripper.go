package http

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/micro/go-micro/selector"
)

type roundTripper struct {
	rt   http.RoundTripper
	st   selector.Strategy
	opts Options
}

func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	s, err := r.opts.Registry.GetService(req.URL.Host)
	if err != nil {
		return nil, err
	}

	next := r.st(s)

	// rudimentary retry 3 times
	for i := 0; i < 3; i++ {
		n, err := next()
		if err != nil {
			continue
		}
		req.URL.Host = fmt.Sprintf("%s:%d", n.Address, n.Port)
		w, err := r.rt.RoundTrip(req)
		if err != nil {
			continue
		}
		return w, nil
	}

	return nil, errors.New("failed request")
}
