package prometheus_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/asim/go-micro/v3/broker"
	bmemory "github.com/asim/go-micro/plugins/broker/memory/v3"
	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/selector"
	"github.com/asim/go-micro/plugins/registry/memory/v3"
	"github.com/asim/go-micro/v3/server"
	promwrapper "github.com/asim/go-micro/plugins/wrapper/monitoring/prometheus/v3"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

type Test interface {
	Method(ctx context.Context, in *TestRequest, opts ...client.CallOption) (*TestResponse, error)
}

type TestRequest struct {
	IsError bool
}
type TestResponse struct{}

type testHandler struct{}

func (t *testHandler) Method(ctx context.Context, req *TestRequest, rsp *TestResponse) error {
	if req.IsError {
		return fmt.Errorf("test error")
	}
	return nil
}

func TestPrometheusMetrics(t *testing.T) {
	// setup
	reg := memory.NewRegistry()
	brk := bmemory.NewBroker(broker.Registry(reg))
	sel := selector.NewSelector(selector.Registry(reg))

	name := "test"
	id := "id-1234567890"
	version := "1.2.3.4"

	c := client.NewClient(client.Selector(sel))
	s := server.NewServer(
		server.Name(name),
		server.Version(version),
		server.Id(id),
		server.Registry(reg),
		server.Broker(brk),
		server.WrapHandler(
			promwrapper.NewHandlerWrapper(
				promwrapper.ServiceName(name),
				promwrapper.ServiceVersion(version),
				promwrapper.ServiceID(id),
			),
		),
	)

	defer s.Stop()

	type Test struct {
		*testHandler
	}

	s.Handle(
		s.NewHandler(&Test{new(testHandler)}),
	)

	if err := s.Start(); err != nil {
		t.Fatalf("Unexpected error starting server: %v", err)
	}

	req := c.NewRequest(name, "Test.Method", &TestRequest{IsError: false}, client.WithContentType("application/json"))
	rsp := TestResponse{}

	assert.NoError(t, c.Call(context.TODO(), req, &rsp))

	req = c.NewRequest(name, "Test.Method", &TestRequest{IsError: true}, client.WithContentType("application/json"))
	assert.Error(t, c.Call(context.TODO(), req, &rsp))

	list, _ := prometheus.DefaultGatherer.Gather()

	metric := findMetricByName(list, dto.MetricType_SUMMARY, "micro_latency_microseconds")

	if metric == nil || metric.Metric == nil || len(metric.Metric) == 0 {
		t.Fatalf("no metrics returned")
	}

	for _, v := range metric.Metric[0].Label {
		switch *v.Name {
		case "micro_version":
			assert.Equal(t, version, *v.Value)
		case "micro_id":
			assert.Equal(t, id, *v.Value)
		case "micro_name":
			assert.Equal(t, name, *v.Value)
		case "micro_endpoint":
			assert.Equal(t, "Test.Method", *v.Value)
		default:
			t.Fatalf("unknown %v with %v", *v.Name, *v.Value)
		}
	}

	assert.Equal(t, uint64(2), *metric.Metric[0].Summary.SampleCount)
	assert.True(t, *metric.Metric[0].Summary.SampleSum > 0)

	metric = findMetricByName(list, dto.MetricType_HISTOGRAM, "micro_request_duration_seconds")

	for _, v := range metric.Metric[0].Label {
		switch *v.Name {
		case "micro_version":
			assert.Equal(t, version, *v.Value)
		case "micro_id":
			assert.Equal(t, id, *v.Value)
		case "micro_name":
			assert.Equal(t, name, *v.Value)
		case "micro_endpoint":
			assert.Equal(t, "Test.Method", *v.Value)
		default:
			t.Fatalf("unknown %v with %v", *v.Name, *v.Value)
		}
	}

	assert.Equal(t, uint64(2), *metric.Metric[0].Histogram.SampleCount)
	assert.True(t, *metric.Metric[0].Histogram.SampleSum > 0)

	metric = findMetricByName(list, dto.MetricType_COUNTER, "micro_request_total")

	for _, v := range metric.Metric[0].Label {
		switch *v.Name {
		case "micro_version":
			assert.Equal(t, version, *v.Value)
		case "micro_id":
			assert.Equal(t, id, *v.Value)
		case "micro_name":
			assert.Equal(t, name, *v.Value)
		case "micro_endpoint":
			assert.Equal(t, "Test.Method", *v.Value)
		case "micro_status":
			assert.Equal(t, "failure", *v.Value)
		}
	}
	assert.Equal(t, *metric.Metric[0].Counter.Value, float64(1))

	for _, v := range metric.Metric[1].Label {
		switch *v.Name {
		case "micro_version":
			assert.Equal(t, version, *v.Value)
		case "micro_id":
			assert.Equal(t, id, *v.Value)
		case "micro_name":
			assert.Equal(t, name, *v.Value)
		case "micro_endpoint":
			assert.Equal(t, "Test.Method", *v.Value)
		case "micro_status":
			assert.Equal(t, "success", *v.Value)
		}
	}

	assert.Equal(t, *metric.Metric[1].Counter.Value, float64(1))
}

func findMetricByName(list []*dto.MetricFamily, tp dto.MetricType, name string) *dto.MetricFamily {
	for _, metric := range list {
		if *metric.Name == name && *metric.Type == tp {
			return metric
		}
	}

	return nil
}
