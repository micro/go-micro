package handlers

import (
	"fmt"
	pb "github.com/asim/go-micro/examples/greeter-service/proto"

	"golang.org/x/net/context"
)

func (h *handlers) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	if in.Name == "world" {
		return nil, fmt.Errorf("No world here")
	}

	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}
