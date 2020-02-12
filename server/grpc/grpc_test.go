package grpc

import (
	"context"
	"testing"

	"github.com/micro/go-micro/v2/errors"
	"github.com/micro/go-micro/v2/registry/memory"
	"github.com/micro/go-micro/v2/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	pb "github.com/micro/go-micro/v2/server/grpc/proto"
)

// server is used to implement helloworld.GreeterServer.
type testServer struct{}

// TestHello implements helloworld.GreeterServer
func (s *testServer) Call(ctx context.Context, req *pb.Request, rsp *pb.Response) error {
	if req.Name == "Error" {
		return &errors.Error{Id: "1", Code: 99, Detail: "detail"}
	}

	rsp.Msg = "Hello " + req.Name
	return nil
}

func TestGRPCServer(t *testing.T) {
	r := memory.NewRegistry()
	s := NewServer(
		server.Name("foo"),
		server.Registry(r),
	)

	pb.RegisterTestHandler(s, &testServer{})

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
