package datadog

import (
	"context"
	"sync"
	"testing"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/selector"
	microerr "github.com/asim/go-micro/v3/errors"
	"github.com/asim/go-micro/plugins/registry/memory/v3"
	"github.com/asim/go-micro/v3/server"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/mocktracer"

	cli "github.com/asim/go-micro/v3/client"
	srv "github.com/asim/go-micro/v3/server"
)

type Test interface {
	Method(ctx context.Context, in *TestRequest, opts ...client.CallOption) (*TestResponse, error)
}

type TestRequest struct {
	IsError bool
}
type TestResponse struct {
	Message string
}

type testHandler struct{}

func (t *testHandler) Method(ctx context.Context, req *TestRequest, rsp *TestResponse) error {
	if req.IsError {
		return microerr.BadRequest("bad", "test error")
	}

	rsp.Message = "passed"

	return nil
}

func TestClient(t *testing.T) {
	// setup
	assert := assert.New(t)
	for name, tt := range map[string]struct {
		message     string
		isError     bool
		wantMessage string
		wantStatus  string
	}{
		"OK": {
			message:     "passed",
			isError:     false,
			wantMessage: "passed",
			wantStatus:  "OK",
		},
		"Invalid": {
			message:     "",
			isError:     true,
			wantMessage: "",
			wantStatus:  "InvalidArgument",
		},
	} {
		t.Run(name, func(t *testing.T) {
			mt := mocktracer.Start()
			defer mt.Stop()

			registry := memory.NewRegistry()
			sel := selector.NewSelector(selector.Registry(registry))

			serverName := "micro.server.name"
			serverID := "id-1234567890"
			serverVersion := "1.0.0"

			c := cli.NewClient(
				client.Selector(sel),
				client.WrapCall(NewCallWrapper()),
			)

			s := srv.NewServer(
				server.Name(serverName),
				server.Version(serverVersion),
				server.Id(serverID),
				server.Registry(registry),
				server.WrapSubscriber(NewSubscriberWrapper()),
				server.WrapHandler(NewHandlerWrapper()),
			)
			defer s.Stop()

			type Test struct {
				*testHandler
			}

			s.Handle(s.NewHandler(&Test{new(testHandler)}))

			if err := s.Start(); err != nil {
				t.Fatalf("Unexpected error starting server: %v", err)
			}

			span, ctx := StartSpanFromContext(context.Background(), "root", tracer.ServiceName("root"), tracer.ResourceName("root"))

			req := c.NewRequest(serverName, "Test.Method", &TestRequest{IsError: tt.isError}, client.WithContentType("application/json"))
			rsp := TestResponse{}
			err := c.Call(ctx, req, &rsp)
			if tt.isError {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
			assert.Equal(rsp.Message, tt.message)

			span.Finish()

			spans := mt.FinishedSpans()
			assert.Len(spans, 3)

			var serverSpan, clientSpan, rootSpan mocktracer.Span
			for _, s := range spans {
				// order of traces in buffer is not garanteed
				switch s.OperationName() {
				case "micro.server":
					serverSpan = s
				case "micro.client":
					clientSpan = s
				case "root":
					rootSpan = s
				}
			}

			assert.NotNil(serverSpan)
			assert.NotNil(clientSpan)
			assert.NotNil(rootSpan)

			assert.Equal(rootSpan.TraceID(), clientSpan.TraceID())
			assert.Equal(serverSpan.Tag(tagStatus), tt.wantStatus)
			assert.Equal("Test.Method", serverSpan.Tag(ext.ResourceName))
			assert.Equal(rootSpan.TraceID(), serverSpan.TraceID())
		})
	}
}

func TestRace(t *testing.T) {
	// setup
	assert := assert.New(t)

	mt := mocktracer.Start()
	defer mt.Stop()

	registry := memory.NewRegistry()
	sel := selector.NewSelector(selector.Registry(registry))

	serverName := "micro.server.name"
	serverID := "id-1234567890"
	serverVersion := "1.0.0"

	c := cli.NewClient(
		client.Selector(sel),
		client.WrapCall(NewCallWrapper()),
	)

	s := srv.NewServer(
		server.Name(serverName),
		server.Version(serverVersion),
		server.Id(serverID),
		server.Registry(registry),
		server.WrapSubscriber(NewSubscriberWrapper()),
		server.WrapHandler(NewHandlerWrapper()),
	)
	defer s.Stop()

	type Test struct {
		*testHandler
	}

	s.Handle(s.NewHandler(&Test{new(testHandler)}))

	if err := s.Start(); err != nil {
		t.Fatalf("Unexpected error starting server: %v", err)
	}

	span, ctx := StartSpanFromContext(context.Background(), "root", tracer.ServiceName("root"), tracer.ResourceName("root"))

	num := 100

	var wg sync.WaitGroup
	wg.Add(num)
	for i := 0; i < num; i++ {
		func() {
			go func(i int) {
				defer wg.Done()

				req := c.NewRequest(serverName, "Test.Method", &TestRequest{IsError: false}, client.WithContentType("application/json"))
				rsp := TestResponse{}
				err := c.Call(ctx, req, &rsp)
				assert.NoError(err)
			}(i)
		}()
	}
	wg.Wait()

	span.Finish()
	spans := mt.FinishedSpans()
	assert.Len(spans, (num*2)+1)
}
