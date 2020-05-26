package grpc

import (
	"context"
	"crypto/tls"
	"sync"
	"testing"
	"time"

	"github.com/micro/go-micro/v2/registry/memory"
	"github.com/micro/go-micro/v2/service"
	hello "github.com/micro/go-micro/v2/service/grpc/proto"
	mls "github.com/micro/go-micro/v2/util/tls"
)

type testHandler struct{}

func (t *testHandler) Call(ctx context.Context, req *hello.Request, rsp *hello.Response) error {
	rsp.Msg = "Hello " + req.Name
	return nil
}

func TestGRPCService(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create memory registry
	r := memory.NewRegistry()

	// create GRPC service
	service := NewService(
		service.Name("test.service"),
		service.Registry(r),
		service.AfterStart(func() error {
			wg.Done()
			return nil
		}),
		service.Context(ctx),
	)

	// register test handler
	hello.RegisterTestHandler(service.Server(), &testHandler{})

	// run service
	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		errCh <- service.Run()
	}()

	// wait for start
	wg.Wait()

	// create client
	test := hello.NewTestService("test.service", service.Client())

	// call service
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Duration(time.Second))
	defer cancel2()
	rsp, err := test.Call(ctx2, &hello.Request{
		Name: "John",
	})
	if err != nil {
		t.Fatal(err)
	}

	// check server
	select {
	case err := <-errCh:
		t.Fatal(err)
	case <-time.After(time.Second):
		break
	}

	// check message
	if rsp.Msg != "Hello John" {
		t.Fatalf("unexpected response %s", rsp.Msg)
	}
}

func TestGRPCTLSService(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create memory registry
	r := memory.NewRegistry()

	// create cert
	cert, err := mls.Certificate("test.service")
	if err != nil {
		t.Fatal(err)
	}
	config := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	// create GRPC service
	service := NewService(
		service.Name("test.service"),
		service.Registry(r),
		service.AfterStart(func() error {
			wg.Done()
			return nil
		}),
		service.Context(ctx),
		// set TLS config
		WithTLS(config),
	)

	// register test handler
	hello.RegisterTestHandler(service.Server(), &testHandler{})

	// run service
	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		errCh <- service.Run()
	}()

	// wait for start
	wg.Wait()

	// create client
	test := hello.NewTestService("test.service", service.Client())

	// call service
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Duration(time.Second))
	defer cancel2()
	rsp, err := test.Call(ctx2, &hello.Request{
		Name: "John",
	})
	if err != nil {
		t.Fatal(err)
	}

	// check server
	select {
	case err := <-errCh:
		t.Fatal(err)
	case <-time.After(time.Second):
		break
	}

	// check message
	if rsp.Msg != "Hello John" {
		t.Fatalf("unexpected response %s", rsp.Msg)
	}
}
