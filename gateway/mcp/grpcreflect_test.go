package mcp

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	helloworld "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/reflection"
)

type reflectedGreeter struct {
	helloworld.UnimplementedGreeterServer
}

func (reflectedGreeter) SayHello(_ context.Context, req *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	return &helloworld.HelloReply{Message: "hello " + req.Name}, nil
}

func TestReflectedGRPCTargetDiscoversAndCallsUnaryTool(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	helloworld.RegisterGreeterServer(grpcServer, reflectedGreeter{})
	reflection.Register(grpcServer)
	go grpcServer.Serve(lis)
	defer grpcServer.Stop()

	s := newTestServer(Options{Context: context.Background()})
	tools, err := s.reflectedGRPCTools(ReflectedGRPCTarget{
		Name:    "demo",
		Address: lis.Addr().String(),
		Timeout: 3 * time.Second,
	})
	if err != nil {
		t.Fatalf("discover reflected tools: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("tools len = %d, want 1", len(tools))
	}
	tool := tools[0]
	if tool.Name != "demo.helloworld_Greeter.SayHello" {
		t.Fatalf("tool name = %q", tool.Name)
	}
	props := tool.InputSchema["properties"].(map[string]interface{})
	if _, ok := props["name"]; !ok {
		t.Fatalf("input schema missing name: %#v", tool.InputSchema)
	}

	out, err := tool.Handler(map[string]interface{}{"name": "Ada"})
	if err != nil {
		t.Fatalf("call reflected tool: %v", err)
	}
	got := out.(map[string]interface{})["message"]
	if got != "hello Ada" {
		t.Fatalf("message = %v, want hello Ada", got)
	}
}
