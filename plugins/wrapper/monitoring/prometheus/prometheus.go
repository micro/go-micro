package prometheus

import (
	"context"
	"fmt"
	"sync"

	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/logger"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/v3/server"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// default metric prefix
	DefaultMetricPrefix = "micro_"
	// default label prefix
	DefaultLabelPrefix = "micro_"

	opsCounter           *prometheus.CounterVec
	timeCounterSummary   *prometheus.SummaryVec
	timeCounterHistogram *prometheus.HistogramVec

	mu sync.Mutex
)

type Options struct {
	Name    string
	Version string
	ID      string
}

type Option func(*Options)

func ServiceName(name string) Option {
	return func(opts *Options) {
		opts.Name = name
	}
}

func ServiceVersion(version string) Option {
	return func(opts *Options) {
		opts.Version = version
	}
}

func ServiceID(id string) Option {
	return func(opts *Options) {
		opts.ID = id
	}
}

func registerMetrics() {
	mu.Lock()
	defer mu.Unlock()

	if opsCounter == nil {
		opsCounter = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("%srequest_total", DefaultMetricPrefix),
				Help: "Requests processed, partitioned by endpoint and status",
			},
			[]string{
				fmt.Sprintf("%s%s", DefaultLabelPrefix, "name"),
				fmt.Sprintf("%s%s", DefaultLabelPrefix, "version"),
				fmt.Sprintf("%s%s", DefaultLabelPrefix, "id"),
				fmt.Sprintf("%s%s", DefaultLabelPrefix, "endpoint"),
				fmt.Sprintf("%s%s", DefaultLabelPrefix, "status"),
			},
		)
	}

	if timeCounterSummary == nil {
		timeCounterSummary = prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: fmt.Sprintf("%slatency_microseconds", DefaultMetricPrefix),
				Help: "Request latencies in microseconds, partitioned by endpoint",
			},
			[]string{
				fmt.Sprintf("%s%s", DefaultLabelPrefix, "name"),
				fmt.Sprintf("%s%s", DefaultLabelPrefix, "version"),
				fmt.Sprintf("%s%s", DefaultLabelPrefix, "id"),
				fmt.Sprintf("%s%s", DefaultLabelPrefix, "endpoint"),
			},
		)
	}

	if timeCounterHistogram == nil {
		timeCounterHistogram = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: fmt.Sprintf("%srequest_duration_seconds", DefaultMetricPrefix),
				Help: "Request time in seconds, partitioned by endpoint",
			},
			[]string{
				fmt.Sprintf("%s%s", DefaultLabelPrefix, "name"),
				fmt.Sprintf("%s%s", DefaultLabelPrefix, "version"),
				fmt.Sprintf("%s%s", DefaultLabelPrefix, "id"),
				fmt.Sprintf("%s%s", DefaultLabelPrefix, "endpoint"),
			},
		)
	}

	for _, collector := range []prometheus.Collector{opsCounter, timeCounterSummary, timeCounterHistogram} {
		if err := prometheus.DefaultRegisterer.Register(collector); err != nil {
			// if already registered, skip fatal
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				logger.Fatal(err)
			}
		}
	}

}

type wrapper struct {
	options  Options
	callFunc client.CallFunc
	client.Client
}

func NewClientWrapper(opts ...Option) client.Wrapper {
	registerMetrics()

	options := Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return func(c client.Client) client.Client {
		handler := &wrapper{
			options: options,
			Client:  c,
		}

		return handler
	}
}

func NewCallWrapper(opts ...Option) client.CallWrapper {
	registerMetrics()

	options := Options{}
	for _, opt := range opts {
		opt(&options)
	}

	return func(fn client.CallFunc) client.CallFunc {
		handler := &wrapper{
			options:  options,
			callFunc: fn,
		}

		return handler.CallFunc
	}
}

func (w *wrapper) CallFunc(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
	endpoint := fmt.Sprintf("%s.%s", req.Service(), req.Endpoint())

	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		us := v * 1000000 // make microseconds
		timeCounterSummary.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint).Observe(us)
		timeCounterHistogram.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint).Observe(v)
	}))
	defer timer.ObserveDuration()

	err := w.callFunc(ctx, node, req, rsp, opts)
	if err == nil {
		opsCounter.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint, "success").Inc()
	} else {
		opsCounter.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint, "failure").Inc()
	}

	return err

}

func (w *wrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	endpoint := fmt.Sprintf("%s.%s", req.Service(), req.Endpoint())

	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		us := v * 1000000 // make microseconds
		timeCounterSummary.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint).Observe(us)
		timeCounterHistogram.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint).Observe(v)
	}))
	defer timer.ObserveDuration()

	err := w.Client.Call(ctx, req, rsp, opts...)
	if err == nil {
		opsCounter.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint, "success").Inc()
	} else {
		opsCounter.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint, "failure").Inc()
	}

	return err
}

func (w *wrapper) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	endpoint := fmt.Sprintf("%s.%s", req.Service(), req.Endpoint())

	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		us := v * 1000000 // make microseconds
		timeCounterSummary.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint).Observe(us)
		timeCounterHistogram.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint).Observe(v)
	}))
	defer timer.ObserveDuration()

	stream, err := w.Client.Stream(ctx, req, opts...)
	if err == nil {
		opsCounter.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint, "success").Inc()
	} else {
		opsCounter.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint, "failure").Inc()
	}

	return stream, err
}

func (w *wrapper) Publish(ctx context.Context, p client.Message, opts ...client.PublishOption) error {
	endpoint := p.Topic()

	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		us := v * 1000000 // make microseconds
		timeCounterSummary.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint).Observe(us)
		timeCounterHistogram.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint).Observe(v)
	}))
	defer timer.ObserveDuration()

	err := w.Client.Publish(ctx, p, opts...)
	if err == nil {
		opsCounter.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint, "success").Inc()
	} else {
		opsCounter.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint, "failure").Inc()
	}

	return err
}

func NewHandlerWrapper(opts ...Option) server.HandlerWrapper {
	registerMetrics()

	options := Options{}
	for _, opt := range opts {
		opt(&options)
	}

	handler := &wrapper{
		options: options,
	}

	return handler.HandlerFunc
}

func (w *wrapper) HandlerFunc(fn server.HandlerFunc) server.HandlerFunc {
	return func(ctx context.Context, req server.Request, rsp interface{}) error {
		endpoint := req.Endpoint()

		timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
			us := v * 1000000 // make microseconds
			timeCounterSummary.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint).Observe(us)
			timeCounterHistogram.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint).Observe(v)
		}))
		defer timer.ObserveDuration()

		err := fn(ctx, req, rsp)
		if err == nil {
			opsCounter.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint, "success").Inc()
		} else {
			opsCounter.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint, "failure").Inc()
		}

		return err
	}
}

func NewSubscriberWrapper(opts ...Option) server.SubscriberWrapper {
	registerMetrics()

	options := Options{}
	for _, opt := range opts {
		opt(&options)
	}

	handler := &wrapper{
		options: options,
	}

	return handler.SubscriberFunc
}

func (w *wrapper) SubscriberFunc(fn server.SubscriberFunc) server.SubscriberFunc {
	return func(ctx context.Context, msg server.Message) error {
		endpoint := msg.Topic()

		timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
			us := v * 1000000 // make microseconds
			timeCounterSummary.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint).Observe(us)
			timeCounterHistogram.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint).Observe(v)
		}))
		defer timer.ObserveDuration()

		err := fn(ctx, msg)
		if err == nil {
			opsCounter.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint, "success").Inc()
		} else {
			opsCounter.WithLabelValues(w.options.Name, w.options.Version, w.options.ID, endpoint, "failure").Inc()
		}

		return err
	}
}
