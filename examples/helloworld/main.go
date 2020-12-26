package main

import (
	"context"
	"log"

	pb "github.com/micro/go-micro/examples/helloworld/proto"
	"github.com/micro/go-micro/v2"
)

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *pb.Request, rsp *pb.Response) error {
	rsp.Greeting = "Hello " + req.Name
	return nil
}

func main() {
	service := micro.NewService(
		micro.Name("helloworld"),
	)

	service.Init()

	pb.RegisterGreeterHandler(service.Server(), new(Greeter))

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
