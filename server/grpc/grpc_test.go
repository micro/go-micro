package grpc_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/micro/go-micro/v2"
	bmemory "github.com/micro/go-micro/v2/broker/memory"
	"github.com/micro/go-micro/v2/client"
	gcli "github.com/micro/go-micro/v2/client/grpc"
	"github.com/micro/go-micro/v2/errors"
	rmemory "github.com/micro/go-micro/v2/registry/memory"
	"github.com/micro/go-micro/v2/server"
	gsrv "github.com/micro/go-micro/v2/server/grpc"
	tgrpc "github.com/micro/go-micro/v2/transport/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	pb "github.com/micro/go-micro/v2/server/grpc/proto"
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
	s := gsrv.NewServer(
		server.Broker(b),
		server.Name("foo"),
		server.Registry(r),
		server.Transport(tr),
	)
	c := gcli.NewClient(
		client.Registry(r),
		client.Broker(b),
		client.Transport(tr),
	)
	ctx := context.TODO()

	h := &testServer{}
	pb.RegisterTestHandler(s, h)

	if err := micro.RegisterSubscriber("test_topic", s, h.Handle); err != nil {
		t.Fatal(err)
	}
	if err := micro.RegisterSubscriber("error_topic", s, h.HandleError); err != nil {
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

	pub := micro.NewEvent("test_topic", c)
	pubErr := micro.NewEvent("error_topic", c)
	cnt := 4
	for i := 0; i < cnt; i++ {
		if err = pub.Publish(ctx, &pb.Request{Name: fmt.Sprintf("msg %d", i)}); err != nil {
			t.Fatal(err)
		}
	}

	if h.msgCount != cnt {
		t.Fatalf("pub/sub not work, or invalid message count %d", h.msgCount)
	}
	if err = pubErr.Publish(ctx, &pb.Request{}); err == nil {
		t.Fatal("this must return error, as we return error from handler")
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
		verr, ok := st.Details()[0].(*errors.Error)
		if !ok {
			t.Fatalf("invalid error received %#+v\n", st.Details()[0])
		}
		if verr.Code != 99 && verr.Id != "1" && verr.Detail != "detail" {
			t.Fatalf("invalid error received %#+v\n", verr)
		}
	}
}
