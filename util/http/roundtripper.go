package http

import (
	"errors"
	"math/rand"
	"net/http"

	"github.com/micro/go-micro/v2/registry"
)

type roundTripper struct {
	rt   http.RoundTripper
	opts Options
}

func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	s, err := r.opts.Registry.GetService(req.URL.Host)
	if err != nil {
		return nil, err
	}

	// get the nodes
	var nodes []*registry.Node
	for _, srv := range s {
		nodes = append(nodes, srv.Nodes...)
	}
	if len(nodes) == 0 {
		return nil, errors.New("no nodes found")
	}

	// rudimentary retry 3 times
	for i := 0; i < 3; i++ {
		// select a random node
		n := nodes[rand.Int()%len(nodes)]

		req.URL.Host = n.Address
		w, err := r.rt.RoundTrip(req)
		if err != nil {
			continue
		}

		return w, nil
	}

	return nil, errors.New("failed request")
}
