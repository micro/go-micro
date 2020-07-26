package grpc_test

import (
	"context"
	"fmt"
	"testing"

	bmemory "github.com/micro/go-micro/v3/broker/memory"
	"github.com/micro/go-micro/v3/client"
	gcli "github.com/micro/go-micro/v3/client/grpc"
	"github.com/micro/go-micro/v3/errors"
	pberr "github.com/micro/go-micro/v3/errors/proto"
	rmemory "github.com/micro/go-micro/v3/registry/memory"
	"github.com/micro/go-micro/v3/router"
	rtreg "github.com/micro/go-micro/v3/router/registry"
	"github.com/micro/go-micro/v3/server"
	gsrv "github.com/micro/go-micro/v3/server/grpc"
	pb "github.com/micro/go-micro/v3/server/grpc/proto"
	tgrpc "github.com/micro/go-micro/v3/transport/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// server is used to implement helloworld.GreeterServer.
type testServer struct {
	msgCount int
}

func (s *testServer) Handle(ctx context.Context, msg *pb.Request) error {
	s.msgCount++
	return nil
}
func (s *testServer) HandleError(ctx context.Context, msg *pb.Request) error {
	return fmt.Errorf("fake")
}

// TestHello implements helloworld.GreeterServer
func (s *testServer) CallPcre(ctx context.Context, req *pb.Request, rsp *pb.Response) error {
	if req.Name == "Error" {
		return &errors.Error{Id: "1", Code: 99, Detail: "detail"}
	}

	rsp.Msg = "Hello " + req.Name
	return nil
}

// TestHello implements helloworld.GreeterServer
func (s *testServer) CallPcreInvalid(ctx context.Context, req *pb.Request, rsp *pb.Response) error {
	if req.Name == "Error" {
		return &errors.Error{Id: "1", Code: 99, Detail: "detail"}
	}

	rsp.Msg = "Hello " + req.Name
	return nil
}

// TestHello implements helloworld.GreeterServer
func (s *testServer) Call(ctx context.Context, req *pb.Request, rsp *pb.Response) error {
	if req.Name == "Error" {
		return &errors.Error{Id: "1", Code: 99, Detail: "detail"}
	}

	if req.Name == "Panic" {
		// make it panic
		panic("handler panic")
	}

	rsp.Msg = "Hello " + req.Name
	return nil
}

/*
func BenchmarkServer(b *testing.B) {
	r := rmemory.NewRegistry()
	br := bmemory.NewBroker()
	tr := tgrpc.NewTransport()
	s := gsrv.NewServer(
		server.Broker(br),
		server.Name("foo"),
		server.Registry(r),
		server.Transport(tr),
	)
	c := gcli.NewClient(
		client.Registry(r),
		client.Broker(br),
		client.Transport(tr),
	)
	ctx := context.TODO()

	h := &testServer{}
	pb.RegisterTestHandler(s, h)
	if err := s.Start(); err != nil {
		b.Fatalf("failed to start: %v", err)
	}

	// check registration
	services, err := r.GetService("foo")
	if err != nil || len(services) == 0 {
		b.Fatalf("failed to get service: %v # %d", err, len(services))
	}

	defer func() {
		if err := s.Stop(); err != nil {
			b.Fatalf("failed to stop: %v", err)
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Call()
	}

}
*/
func TestGRPCServer(t *testing.T) {
	r := rmemory.NewRegistry()
	b := bmemory.NewBroker()
	tr := tgrpc.NewTransport()
	rtr := rtreg.NewRouter(router.Registry(r))

	s := gsrv.NewServer(
		server.Broker(b),
		server.Name("foo"),
		server.Registry(r),
		server.Transport(tr),
	)

	c := gcli.NewClient(
		client.Router(rtr),
		client.Broker(b),
		client.Transport(tr),
	)
	ctx := context.TODO()

	h := &testServer{}
	pb.RegisterTestHandler(s, h)

	if err := s.Subscribe(s.NewSubscriber("test_topic", h.Handle)); err != nil {
		t.Fatal(err)
	}

	if err := s.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// check registration
	services, err := r.GetService("foo")
	if err != nil || len(services) == 0 {
		t.Fatalf("failed to get service: %v # %d", err, len(services))
	}

	defer func() {
		if err := s.Stop(); err != nil {
			t.Fatalf("failed to stop: %v", err)
		}
	}()

	cnt := 4
	for i := 0; i < cnt; i++ {
		msg := c.NewMessage("test_topic", &pb.Request{Name: fmt.Sprintf("msg %d", i)})
		if err = c.Publish(ctx, msg); err != nil {
			t.Fatal(err)
		}
	}

	if h.msgCount != cnt {
		t.Fatalf("pub/sub not work, or invalid message count %d", h.msgCount)
	}

	cc, err := grpc.Dial(s.Options().Address, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to dial server: %v", err)
	}

	testMethods := []string{"/test.Test/Call", "/go.micro.test.Test/Call"}

	for _, method := range testMethods {
		rsp := pb.Response{}

		if err := cc.Invoke(context.Background(), method, &pb.Request{Name: "John"}, &rsp); err != nil {
			t.Fatalf("error calling server: %v", err)
		}

		if rsp.Msg != "Hello John" {
			t.Fatalf("Got unexpected response %v", rsp.Msg)
		}
	}

	// Test grpc error
	rsp := pb.Response{}

	if err := cc.Invoke(context.Background(), "/test.Test/Call", &pb.Request{Name: "Error"}, &rsp); err != nil {
		st, ok := status.FromError(err)
		if !ok {
			t.Fatalf("invalid error received %#+v\n", err)
		}
		verr, ok := st.Details()[0].(*pberr.Error)
		if !ok {
			t.Fatalf("invalid error received %#+v\n", st.Details()[0])
		}
		if verr.Code != 99 && verr.Id != "1" && verr.Detail != "detail" {
			t.Fatalf("invalid error received %#+v\n", verr)
		}
	}
}

// TestGRPCServerWithPanicWrapper test grpc server with panic wrapper
// gRPC server should not crash when wrapper crashed
func TestGRPCServerWithPanicWrapper(t *testing.T) {
	r := rmemory.NewRegistry()
	b := bmemory.NewBroker()
	tr := tgrpc.NewTransport()
	s := gsrv.NewServer(
		server.Broker(b),
		server.Name("foo"),
		server.Registry(r),
		server.Transport(tr),
		server.WrapHandler(func(hf server.HandlerFunc) server.HandlerFunc {
			return func(ctx context.Context, req server.Request, rsp interface{}) error {
				// make it panic
				panic("wrapper panic")
			}
		}),
	)

	h := &testServer{}
	pb.RegisterTestHandler(s, h)

	if err := s.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// check registration
	services, err := r.GetService("foo")
	if err != nil || len(services) == 0 {
		t.Fatalf("failed to get service: %v # %d", err, len(services))
	}

	defer func() {
		if err := s.Stop(); err != nil {
			t.Fatalf("failed to stop: %v", err)
		}
	}()

	cc, err := grpc.Dial(s.Options().Address, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to dial server: %v", err)
	}

	rsp := pb.Response{}
	if err := cc.Invoke(context.Background(), "/test.Test/Call", &pb.Request{Name: "John"}, &rsp); err == nil {
		t.Fatal("this must return error, as wrapper should be panic")
	}

	// both wrapper and handler should panic
	rsp = pb.Response{}
	if err := cc.Invoke(context.Background(), "/test.Test/Call", &pb.Request{Name: "Panic"}, &rsp); err == nil {
		t.Fatal("this must return error, as wrapper and handler should be panic")
	}
}

// TestGRPCServerWithPanicWrapper test grpc server with panic handler
// gRPC server should not crash when handler crashed
func TestGRPCServerWithPanicHandler(t *testing.T) {
	r := rmemory.NewRegistry()
	b := bmemory.NewBroker()
	tr := tgrpc.NewTransport()
	s := gsrv.NewServer(
		server.Broker(b),
		server.Name("foo"),
		server.Registry(r),
		server.Transport(tr),
	)

	h := &testServer{}
	pb.RegisterTestHandler(s, h)

	if err := s.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// check registration
	services, err := r.GetService("foo")
	if err != nil || len(services) == 0 {
		t.Fatalf("failed to get service: %v # %d", err, len(services))
	}

	defer func() {
		if err := s.Stop(); err != nil {
			t.Fatalf("failed to stop: %v", err)
		}
	}()

	cc, err := grpc.Dial(s.Options().Address, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to dial server: %v", err)
	}

	rsp := pb.Response{}
	if err := cc.Invoke(context.Background(), "/test.Test/Call", &pb.Request{Name: "Panic"}, &rsp); err == nil {
		t.Fatal("this must return error, as handler should be panic")
	}
}
