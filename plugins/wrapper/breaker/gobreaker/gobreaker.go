package gobreaker

import (
	"context"
	"sync"

	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/errors"
	"github.com/sony/gobreaker"
)

type BreakerMethod int

const (
	BreakService BreakerMethod = iota
	BreakServiceEndpoint
)

type clientWrapper struct {
	bs  gobreaker.Settings
	bm  BreakerMethod
	cbs map[string]*gobreaker.TwoStepCircuitBreaker
	mu  sync.Mutex
	client.Client
}

func (c *clientWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	var svc string

	switch c.bm {
	case BreakService:
		svc = req.Service()
	case BreakServiceEndpoint:
		svc = req.Service() + "." + req.Endpoint()
	}

	c.mu.Lock()
	cb, ok := c.cbs[svc]
	if !ok {
		cb = gobreaker.NewTwoStepCircuitBreaker(c.bs)
		c.cbs[svc] = cb
	}
	c.mu.Unlock()

	cbAllow, err := cb.Allow()
	if err != nil {
		return errors.New(req.Service(), err.Error(), 502)
	}

	if err = c.Client.Call(ctx, req, rsp, opts...); err == nil {
		cbAllow(true)
		return nil
	}

	merr := errors.Parse(err.Error())
	switch {
	case merr.Code == 0:
		merr.Code = 503
	case len(merr.Id) == 0:
		merr.Id = req.Service()
	}

	if merr.Code >= 500 {
		cbAllow(false)
	} else {
		cbAllow(true)
	}

	return merr
}

// NewClientWrapper returns a client Wrapper.
func NewClientWrapper() client.Wrapper {
	return func(c client.Client) client.Client {
		w := &clientWrapper{}
		w.bs = gobreaker.Settings{}
		w.cbs = make(map[string]*gobreaker.TwoStepCircuitBreaker)
		w.Client = c
		return w
	}
}

// NewCustomClientWrapper takes a gobreaker.Settings and BreakerMethod. Returns a client Wrapper.
func NewCustomClientWrapper(bs gobreaker.Settings, bm BreakerMethod) client.Wrapper {
	return func(c client.Client) client.Client {
		w := &clientWrapper{}
		w.bm = bm
		w.bs = bs
		w.cbs = make(map[string]*gobreaker.TwoStepCircuitBreaker)
		w.Client = c
		return w
	}
}
