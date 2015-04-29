package main

import (
	"os"

	"github.com/asim/go-micro/client"
	log "github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/asim/go-micro/examples/greeter-service/proto"
)

const (
	defaultName = "world"
)

func main() {
	name := defaultName
	if len(os.Args) > 1 {
		name = os.Args[1]
	}

	// Create new request to service go.micro.service.go-template
	var r *pb.HelloReply
	err := client.NewRequest("go.micro.service.greeter", client.GRPCRequest(func(cc *grpc.ClientConn) (err error) {
		c := pb.NewGreeterClient(cc)
		r, err = c.SayHello(context.Background(), &pb.HelloRequest{Name: name})
		return
	}))
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}

	log.Infof("Greeting: %s", r.Message)

}
