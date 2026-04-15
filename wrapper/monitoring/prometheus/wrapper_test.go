package prometheus

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/codec"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/server"
)

// mockServerRequest is a minimal implementation of server.Request.
type mockServerRequest struct {
	service  string
	endpoint string
}

func (r *mockServerRequest) Service() string           { return r.service }
func (r *mockServerRequest) Method() string            { return r.endpoint }
func (r *mockServerRequest) Endpoint() string          { return r.endpoint }
func (r *mockServerRequest) ContentType() string       { return "" }
func (r *mockServerRequest) Header() map[string]string { return nil }
func (r *mockServerRequest) Body() interface{}         { return nil }
func (r *mockServerRequest) Read() ([]byte, error)     { return nil, nil }
func (r *mockServerRequest) Codec() codec.Reader       { return nil }
func (r *mockServerRequest) Stream() bool              { return false }

// mockMessage implements server.Message for subscriber tests.
type mockMessage struct {
	topic string
}

func (m *mockMessage) Topic() string             { return m.topic }
func (m *mockMessage) Payload() interface{}      { return nil }
func (m *mockMessage) ContentType() string       { return "" }
func (m *mockMessage) Header() map[string]string { return nil }
func (m *mockMessage) Body() []byte              { return nil }
func (m *mockMessage) Codec() codec.Reader       { return nil }

// mockClientRequest is a minimal implementation of client.Request.
type mockClientRequest struct {
	service  string
	endpoint string
}

func (r *mockClientRequest) Service() string     { return r.service }
func (r *mockClientRequest) Method() string      { return r.endpoint }
func (r *mockClientRequest) Endpoint() string    { return r.endpoint }
func (r *mockClientRequest) ContentType() string { return "" }
func (r *mockClientRequest) Body() interface{}   { return nil }
func (r *mockClientRequest) Codec() codec.Writer { return nil }
func (r *mockClientRequest) Stream() bool        { return false }

// isolatedOpts returns wrapper options pinned to a fresh registry so each
// test starts with its own counters. We also vary Name so cached metrics
// bundles don't bleed between tests.
func isolatedOpts(name string) []Option {
	reg := prometheus.NewRegistry()
	return []Option{ServiceName(name), Registerer(reg)}
}

func counterValue(t *testing.T, vec *prometheus.CounterVec, labels ...string) float64 {
	t.Helper()
	c, err := vec.GetMetricWithLabelValues(labels...)
	if err != nil {
		t.Fatalf("GetMetricWithLabelValues: %v", err)
	}
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		t.Fatalf("Write: %v", err)
	}
	return m.GetCounter().GetValue()
}

func histogramCount(t *testing.T, vec *prometheus.HistogramVec, labels ...string) uint64 {
	t.Helper()
	obs, err := vec.GetMetricWithLabelValues(labels...)
	if err != nil {
		t.Fatalf("GetMetricWithLabelValues: %v", err)
	}
	h, ok := obs.(prometheus.Histogram)
	if !ok {
		t.Fatalf("expected Histogram, got %T", obs)
	}
	var m dto.Metric
	if err := h.Write(&m); err != nil {
		t.Fatalf("Write: %v", err)
	}
	return m.GetHistogram().GetSampleCount()
}

func TestHandlerWrapperSuccess(t *testing.T) {
	opts := isolatedOpts("test_handler_success")
	wrap := NewHandlerWrapper(opts...)

	called := false
	handler := wrap(func(ctx context.Context, req server.Request, rsp interface{}) error {
		called = true
		return nil
	})

	err := handler(context.Background(), &mockServerRequest{service: "svc", endpoint: "Foo.Bar"}, nil)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if !called {
		t.Fatal("inner handler was not called")
	}

	m := getMetrics(newOptions(opts...))
	if got := counterValue(t, m.requestTotal, "svc", "Foo.Bar", "success"); got != 1 {
		t.Errorf("success counter = %v, want 1", got)
	}
	if got := histogramCount(t, m.requestDuration, "svc", "Foo.Bar", "success"); got != 1 {
		t.Errorf("histogram count = %v, want 1", got)
	}
}

func TestHandlerWrapperFailure(t *testing.T) {
	opts := isolatedOpts("test_handler_failure")
	wrap := NewHandlerWrapper(opts...)

	boom := errors.New("boom")
	handler := wrap(func(ctx context.Context, req server.Request, rsp interface{}) error {
		return boom
	})

	err := handler(context.Background(), &mockServerRequest{service: "svc", endpoint: "Foo.Bar"}, nil)
	if !errors.Is(err, boom) {
		t.Fatalf("error not propagated, got %v", err)
	}

	m := getMetrics(newOptions(opts...))
	if got := counterValue(t, m.requestTotal, "svc", "Foo.Bar", "fail"); got != 1 {
		t.Errorf("fail counter = %v, want 1", got)
	}
}

func TestSubscriberWrapper(t *testing.T) {
	opts := isolatedOpts("test_subscriber")
	wrap := NewSubscriberWrapper(opts...)

	sub := wrap(func(ctx context.Context, msg server.Message) error {
		return nil
	})

	if err := sub(context.Background(), &mockMessage{topic: "events"}); err != nil {
		t.Fatalf("subscriber returned error: %v", err)
	}

	m := getMetrics(newOptions(opts...))
	if got := counterValue(t, m.requestTotal, "subscriber", "events", "success"); got != 1 {
		t.Errorf("subscriber counter = %v, want 1", got)
	}
}

func TestCallWrapperSuccess(t *testing.T) {
	opts := isolatedOpts("test_call")
	wrap := NewCallWrapper(opts...)

	cf := wrap(func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
		return nil
	})

	err := cf(context.Background(), &registry.Node{}, &mockClientRequest{service: "svc", endpoint: "Foo.Bar"}, nil, client.CallOptions{})
	if err != nil {
		t.Fatalf("call returned error: %v", err)
	}

	m := getMetrics(newOptions(opts...))
	if got := counterValue(t, m.requestTotal, "svc", "Foo.Bar", "success"); got != 1 {
		t.Errorf("call counter = %v, want 1", got)
	}
}

func TestRegisterAlreadyRegistered(t *testing.T) {
	// Creating two wrappers against the same Registerer with the same
	// options must not panic, and the second one must reuse the already
	// registered collector.
	reg := prometheus.NewRegistry()
	opts := []Option{ServiceName("test_dup"), Registerer(reg)}

	_ = NewHandlerWrapper(opts...)
	_ = NewHandlerWrapper(opts...)

	// After resetting the cache we must still not panic, which proves the
	// AlreadyRegisteredError branch in register() is exercised.
	metricsMu.Lock()
	delete(metricsCache, "test_dup\x00\x00")
	metricsMu.Unlock()

	_ = NewHandlerWrapper(opts...)
}

func TestStatusHelper(t *testing.T) {
	if status(nil) != "success" {
		t.Error("nil error should yield success")
	}
	if status(errors.New("x")) != "fail" {
		t.Error("non-nil error should yield fail")
	}
}
