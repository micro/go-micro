package victoriametrics

import (
	"context"
	"fmt"
	"strings"
	"time"

	metrics "github.com/VictoriaMetrics/metrics"
	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/v3/server"
)

var (
	// default metric prefix
	DefaultMetricPrefix = "micro_"
	// default label prefix
	DefaultLabelPrefix = "micro_"
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

func getName(name string, labels []string) string {
	if len(labels) > 0 {
		return fmt.Sprintf(`%s%s{%s}`, DefaultMetricPrefix, name, strings.Join(labels, ","))
	}
	return fmt.Sprintf(`%s%s`, DefaultMetricPrefix, name)
}

func getLabels(opts ...Option) []string {
	options := Options{}

	for _, opt := range opts {
		opt(&options)
	}

	labels := make([]string, 0, 3)
	labels = append(labels, fmt.Sprintf(`%sname="%s"`, DefaultLabelPrefix, options.Name))
	labels = append(labels, fmt.Sprintf(`%sversion="%s"`, DefaultLabelPrefix, options.Version))
	labels = append(labels, fmt.Sprintf(`%sid="%s"`, DefaultLabelPrefix, options.ID))

	return labels
}

type wrapper struct {
	options  Options
	callFunc client.CallFunc
	client.Client
	labels []string
}

func NewClientWrapper(opts ...Option) client.Wrapper {
	labels := getLabels(opts...)

	return func(c client.Client) client.Client {
		handler := &wrapper{
			labels: labels,
			Client: c,
		}

		return handler
	}
}

func NewCallWrapper(opts ...Option) client.CallWrapper {
	labels := getLabels(opts...)

	return func(fn client.CallFunc) client.CallFunc {
		handler := &wrapper{
			labels:   labels,
			callFunc: fn,
		}

		return handler.CallFunc
	}
}

func (w *wrapper) CallFunc(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
	endpoint := fmt.Sprintf("%s.%s", req.Service(), req.Endpoint())
	wlabels := append(w.labels, fmt.Sprintf(`%sendpoint="%s"`, DefaultLabelPrefix, endpoint))

	timeCounterSummary := metrics.GetOrCreateSummary(getName("upstream_latency_seconds", wlabels))
	timeCounterHistogram := metrics.GetOrCreateSummary(getName("request_duration_seconds", wlabels))

	ts := time.Now()
	err := w.callFunc(ctx, node, req, rsp, opts)
	te := time.Since(ts)

	timeCounterSummary.Update(float64(te.Seconds()))
	timeCounterHistogram.Update(te.Seconds())
	if err == nil {
		metrics.GetOrCreateCounter(getName("request_total", append(wlabels, fmt.Sprintf(`%sstatus="success"`, DefaultLabelPrefix)))).Inc()
	} else {
		metrics.GetOrCreateCounter(getName("request_total", append(wlabels, fmt.Sprintf(`%sstatus="failure"`, DefaultLabelPrefix)))).Inc()
	}

	return err
}

func (w *wrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	endpoint := fmt.Sprintf("%s.%s", req.Service(), req.Endpoint())
	wlabels := append(w.labels, fmt.Sprintf(`%sendpoint="%s"`, DefaultLabelPrefix, endpoint))

	timeCounterSummary := metrics.GetOrCreateSummary(getName("upstream_latency_seconds", wlabels))
	timeCounterHistogram := metrics.GetOrCreateSummary(getName("request_duration_seconds", wlabels))

	ts := time.Now()
	err := w.Client.Call(ctx, req, rsp, opts...)
	te := time.Since(ts)

	timeCounterSummary.Update(float64(te.Seconds()))
	timeCounterHistogram.Update(te.Seconds())
	if err == nil {
		metrics.GetOrCreateCounter(getName("request_total", append(wlabels, fmt.Sprintf(`%sstatus="success"`, DefaultLabelPrefix)))).Inc()
	} else {
		metrics.GetOrCreateCounter(getName("request_total", append(wlabels, fmt.Sprintf(`%sstatus="failure"`, DefaultLabelPrefix)))).Inc()
	}

	return err
}

func (w *wrapper) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	endpoint := fmt.Sprintf("%s.%s", req.Service(), req.Endpoint())
	wlabels := append(w.labels, fmt.Sprintf(`%sendpoint="%s"`, DefaultLabelPrefix, endpoint))

	timeCounterSummary := metrics.GetOrCreateSummary(getName("upstream_latency_seconds", wlabels))
	timeCounterHistogram := metrics.GetOrCreateSummary(getName("request_duration_seconds", wlabels))

	ts := time.Now()
	stream, err := w.Client.Stream(ctx, req, opts...)
	te := time.Since(ts)

	timeCounterSummary.Update(float64(te.Seconds()))
	timeCounterHistogram.Update(te.Seconds())
	if err == nil {
		metrics.GetOrCreateCounter(getName("request_total", append(wlabels, fmt.Sprintf(`%sstatus="success"`, DefaultLabelPrefix)))).Inc()
	} else {
		metrics.GetOrCreateCounter(getName("request_total", append(wlabels, fmt.Sprintf(`%sstatus="failure"`, DefaultLabelPrefix)))).Inc()
	}

	return stream, err
}

func (w *wrapper) Publish(ctx context.Context, p client.Message, opts ...client.PublishOption) error {
	endpoint := p.Topic()
	wlabels := append(w.labels, fmt.Sprintf(`%sendpoint="%s"`, DefaultLabelPrefix, endpoint))

	timeCounterSummary := metrics.GetOrCreateSummary(getName("upstream_latency_seconds", wlabels))
	timeCounterHistogram := metrics.GetOrCreateSummary(getName("request_duration_seconds", wlabels))

	ts := time.Now()
	err := w.Client.Publish(ctx, p, opts...)
	te := time.Since(ts)

	timeCounterSummary.Update(float64(te.Seconds()))
	timeCounterHistogram.Update(te.Seconds())
	if err == nil {
		metrics.GetOrCreateCounter(getName("request_total", append(wlabels, fmt.Sprintf(`%sstatus="success"`, DefaultLabelPrefix)))).Inc()
	} else {
		metrics.GetOrCreateCounter(getName("request_total", append(wlabels, fmt.Sprintf(`%sstatus="failure"`, DefaultLabelPrefix)))).Inc()
	}

	return err
}

func NewHandlerWrapper(opts ...Option) server.HandlerWrapper {
	labels := getLabels(opts...)

	handler := &wrapper{
		labels: labels,
	}

	return handler.HandlerFunc
}

func (w *wrapper) HandlerFunc(fn server.HandlerFunc) server.HandlerFunc {
	return func(ctx context.Context, req server.Request, rsp interface{}) error {
		endpoint := req.Endpoint()
		wlabels := append(w.labels, fmt.Sprintf(`%sendpoint="%s"`, DefaultLabelPrefix, endpoint))

		timeCounterSummary := metrics.GetOrCreateSummary(getName("upstream_latency_seconds", wlabels))
		timeCounterHistogram := metrics.GetOrCreateSummary(getName("request_duration_seconds", wlabels))

		ts := time.Now()
		err := fn(ctx, req, rsp)
		te := time.Since(ts)

		timeCounterSummary.Update(float64(te.Seconds()))
		timeCounterHistogram.Update(te.Seconds())
		if err == nil {
			metrics.GetOrCreateCounter(getName("request_total", append(wlabels, fmt.Sprintf(`%sstatus="success"`, DefaultLabelPrefix)))).Inc()
		} else {
			metrics.GetOrCreateCounter(getName("request_total", append(wlabels, fmt.Sprintf(`%sstatus="failure"`, DefaultLabelPrefix)))).Inc()
		}

		return err
	}
}

func NewSubscriberWrapper(opts ...Option) server.SubscriberWrapper {
	labels := getLabels(opts...)

	handler := &wrapper{
		labels: labels,
	}

	return handler.SubscriberFunc
}

func (w *wrapper) SubscriberFunc(fn server.SubscriberFunc) server.SubscriberFunc {
	return func(ctx context.Context, msg server.Message) error {
		endpoint := msg.Topic()
		wlabels := append(w.labels, fmt.Sprintf(`%sendpoint="%s"`, DefaultLabelPrefix, endpoint))

		timeCounterSummary := metrics.GetOrCreateSummary(getName("upstream_latency_seconds", wlabels))
		timeCounterHistogram := metrics.GetOrCreateSummary(getName("request_duration_seconds", wlabels))

		ts := time.Now()
		err := fn(ctx, msg)
		te := time.Since(ts)

		timeCounterSummary.Update(float64(te.Seconds()))
		timeCounterHistogram.Update(te.Seconds())
		if err == nil {
			metrics.GetOrCreateCounter(getName("request_total", append(wlabels, fmt.Sprintf(`%sstatus="success"`, DefaultLabelPrefix)))).Inc()
		} else {
			metrics.GetOrCreateCounter(getName("request_total", append(wlabels, fmt.Sprintf(`%sstatus="failure"`, DefaultLabelPrefix)))).Inc()
		}

		return err
	}
}
