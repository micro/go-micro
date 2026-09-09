package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

func testPool(t *testing.T, size int, ttl time.Duration, idle int, ms int) {
	// setup server
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Errorf("failed to listen: %v", err)
	}
	defer l.Close()

	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &greeterServer{})

	go s.Serve(l)
	defer s.Stop()

	// zero pool
	p := newPool(size, ttl, idle, ms)

	for i := 0; i < 10; i++ {
		// get a conn
		cc, err := p.getConn(context.TODO(), l.Addr().String(), grpc.WithInsecure())
		if err != nil {
			t.Fatal(err)
		}

		rsp := pb.HelloReply{}

		err = cc.Invoke(context.TODO(), "/helloworld.Greeter/SayHello", &pb.HelloRequest{Name: "John"}, &rsp)
		if err != nil {
			t.Fatal(err)
		}

		if rsp.Message != "Hello John" {
			t.Errorf("Got unexpected response %v", rsp.Message)
		}

		// release the conn
		p.release(l.Addr().String(), cc, nil)

		p.Lock()
		if i := p.conns[l.Addr().String()].count; i > size {
			p.Unlock()
			t.Errorf("pool size %d is greater than expected %d", i, size)
		}
		p.Unlock()
	}
}

func TestGRPCPool(t *testing.T) {
	testPool(t, 0, time.Minute, 10, 2)
	testPool(t, 2, time.Minute, 10, 1)
}
