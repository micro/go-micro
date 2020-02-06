package grpc

import (
	"context"
	"net"
	"testing"

	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/client/selector"
	"github.com/micro/go-micro/v2/errors"
	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/registry/memory"
	pgrpc "google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

// server is used to implement helloworld.GreeterServer.
type greeterServer struct{}

// SayHello implements helloworld.GreeterServer
func (g *greeterServer) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	if in.Name == "Error" {
		return nil, &errors.Error{Id: "1", Code: 99, Detail: "detail"}
	}
	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

func TestGRPCClient(t *testing.T) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	s := pgrpc.NewServer()
	pb.RegisterGreeterServer(s, &greeterServer{})

	go s.Serve(l)
	defer s.Stop()

	// create mock registry
	r := memory.NewRegistry()

	// register service
	r.Register(&registry.Service{
		Name:    "helloworld",
		Version: "test",
		Nodes: []*registry.Node{
			{
				Id:      "test-1",
				Address: l.Addr().String(),
				Metadata: map[string]string{
					"protocol": "grpc",
				},
			},
		},
	})

	// create selector
	se := selector.NewSelector(
		selector.Registry(r),
	)

	// create client
	c := NewClient(
		client.Registry(r),
		client.Selector(se),
	)

	testMethods := []string{
		"/helloworld.Greeter/SayHello",
		"Greeter.SayHello",
	}

	for _, method := range testMethods {
		req := c.NewRequest("helloworld", method, &pb.HelloRequest{
			Name: "John",
		})

		rsp := pb.HelloReply{}

		err = c.Call(context.TODO(), req, &rsp)
		if err != nil {
			t.Fatal(err)
		}

		if rsp.Message != "Hello John" {
			t.Fatalf("Got unexpected response %v", rsp.Message)
		}
	}

	req := c.NewRequest("helloworld", "/helloworld.Greeter/SayHello", &pb.HelloRequest{
		Name: "Error",
	})

	rsp := pb.HelloReply{}

	err = c.Call(context.TODO(), req, &rsp)
	if err == nil {
		t.Fatal("nil error received")
	}

	verr, ok := err.(*errors.Error)
	if !ok {
		t.Fatalf("invalid error received %#+v\n", err)
	}

	if verr.Code != 99 && verr.Id != "1" && verr.Detail != "detail" {
		t.Fatalf("invalid error received %#+v\n", verr)
	}

}
