package victoriametrics

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	metrics "github.com/VictoriaMetrics/metrics"
	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/selector"
	"github.com/asim/go-micro/plugins/registry/memory/v3"
	"github.com/asim/go-micro/v3/server"
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

func TestVictoriametrics(t *testing.T) {
	// setup
	registry := memory.NewRegistry()
	sel := selector.NewSelector(selector.Registry(registry))

	name := "test"
	id := "id-1234567890"
	version := "1.2.3.4"

	c := client.NewClient(client.Selector(sel))
	s := server.NewServer(
		server.Name(name),
		server.Version(version),
		server.Id(id),
		server.Registry(registry),
		server.WrapHandler(
			NewHandlerWrapper(
				ServiceName(name),
				ServiceVersion(version),
				ServiceID(id),
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

	buf := bytes.NewBuffer(nil)
	metrics.WritePrometheus(buf, false)

	metric, err := findMetricByName(buf, "sum", "micro_request_total")
	if err != nil {
		t.Fatal(err)
	}

	labels := metric[0]["labels"].(map[string]string)
	for k, v := range labels {
		switch k {
		case "micro_version":
			assert.Equal(t, version, v)
		case "micro_id":
			assert.Equal(t, id, v)
		case "micro_name":
			assert.Equal(t, name, v)
		case "micro_endpoint":
			assert.Equal(t, "Test.Method", v)
		case "micro_status":
			continue
		default:
			t.Fatalf("unknown %v with %v", k, v)
		}
	}
}

func findMetricByName(buf io.Reader, tp string, name string) ([]map[string]interface{}, error) {
	var metrics []map[string]interface{}
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		txt := scanner.Text()
		if strings.HasPrefix(txt, name) {
			mt := make(map[string]interface{})
			v := txt[strings.LastIndex(txt, " "):]
			k := ""
			if idx := strings.Index(txt, "{"); idx > 0 {
				labels := make(map[string]string)
				lb := strings.Split(txt[idx+1:strings.Index(txt, "}")], ",")
				for _, l := range lb {
					p := strings.Split(l, "=")
					labels[strings.Trim(p[0], `"`)] = strings.Trim(p[1], `"`)
				}
				mt["labels"] = labels
				k = txt[:idx]
			} else {
				k = txt[:strings.Index(txt, " ")]
			}
			mt["name"] = k
			mt["value"] = v
			metrics = append(metrics, mt)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(metrics) == 0 {
		return nil, fmt.Errorf("%s %s not found", tp, name)
	}
	return metrics, nil
}
