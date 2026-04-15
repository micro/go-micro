package prometheus

import (
	"context"
	"time"

	"go-micro.dev/v5/client"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/server"
)

// NewHandlerWrapper returns a server.HandlerWrapper that records Prometheus
// metrics (request count and latency) for every incoming RPC handled by the
// server.
func NewHandlerWrapper(opts ...Option) server.HandlerWrapper {
	m := getMetrics(newOptions(opts...))
	return func(fn server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			start := time.Now()
			err := fn(ctx, req, rsp)
			observe(m, req.Service(), req.Endpoint(), err, start)
			return err
		}
	}
}

// NewSubscriberWrapper returns a server.SubscriberWrapper that records
// Prometheus metrics for every message delivered to a subscriber.
func NewSubscriberWrapper(opts ...Option) server.SubscriberWrapper {
	m := getMetrics(newOptions(opts...))
	return func(fn server.SubscriberFunc) server.SubscriberFunc {
		return func(ctx context.Context, msg server.Message) error {
			start := time.Now()
			err := fn(ctx, msg)
			observe(m, "subscriber", msg.Topic(), err, start)
			return err
		}
	}
}

// NewCallWrapper returns a client.CallWrapper that records Prometheus
// metrics for every outgoing RPC issued by the client.
func NewCallWrapper(opts ...Option) client.CallWrapper {
	m := getMetrics(newOptions(opts...))
	return func(fn client.CallFunc) client.CallFunc {
		return func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
			start := time.Now()
			err := fn(ctx, node, req, rsp, opts)
			observe(m, req.Service(), req.Endpoint(), err, start)
			return err
		}
	}
}

// NewClientWrapper returns a client.Wrapper that records Prometheus metrics
// for every Call and Publish issued on the wrapped client. Use it when you
// need metrics for Publish as well as Call; NewCallWrapper only covers Call.
func NewClientWrapper(opts ...Option) client.Wrapper {
	m := getMetrics(newOptions(opts...))
	return func(c client.Client) client.Client {
		return &clientWrapper{Client: c, metrics: m}
	}
}

type clientWrapper struct {
	client.Client
	metrics *metrics
}

func (w *clientWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	start := time.Now()
	err := w.Client.Call(ctx, req, rsp, opts...)
	observe(w.metrics, req.Service(), req.Endpoint(), err, start)
	return err
}

func (w *clientWrapper) Publish(ctx context.Context, p client.Message, opts ...client.PublishOption) error {
	start := time.Now()
	err := w.Client.Publish(ctx, p, opts...)
	observe(w.metrics, "publisher", p.Topic(), err, start)
	return err
}

// observe records a single request into the counter and histogram.
func observe(m *metrics, service, endpoint string, err error, start time.Time) {
	st := status(err)
	m.requestTotal.WithLabelValues(service, endpoint, st).Inc()
	m.requestDuration.WithLabelValues(service, endpoint, st).Observe(time.Since(start).Seconds())
}
