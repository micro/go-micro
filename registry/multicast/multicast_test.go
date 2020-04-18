package multicast_test

import (
	"context"
	"testing"
	"time"

	"github.com/micro/go-micro/v2/client"
	gcli "github.com/micro/go-micro/v2/client/grpc"
	rmcast "github.com/micro/go-micro/v2/registry/multicast"
	"github.com/micro/go-micro/v2/server"
	gsrv "github.com/micro/go-micro/v2/server/grpc"
	pb "github.com/micro/go-micro/v2/server/grpc/proto"
)

// server is used to implement helloworld.GreeterServer.
type testServer struct {
	msgCount int
}

// TestHello implements helloworld.GreeterServer
func (s *testServer) Call(ctx context.Context, req *pb.Request, rsp *pb.Response) error {
	rsp.Msg = "Hello " + req.Uuid
	return nil
}

func initial(t *testing.T) (server.Server, client.Client) {
	r := rmcast.NewRegistry()
	if err := r.Init(); err != nil {
		t.Fatal(err)
	}
	// create a new client
	s := gsrv.NewServer(
		server.Name("foo"),
		server.Registry(r),
		server.RegisterTTL(10*time.Second),
		server.RegisterInterval(3*time.Second),
	)

	// create a new server
	c := gcli.NewClient(
		client.Registry(r),
	)

	h := &testServer{}
	pb.RegisterTestHandler(s, h)

	if err := s.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	return s, c
}

func TestRegistry(t *testing.T) {
	s, c := initial(t)
	defer s.Stop()
	_ = c
	select {}
}
