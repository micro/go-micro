package main

import (
	"context"
	"fmt"

	proto "github.com/asim/go-micro/examples/v4/service/proto"
	"go-micro.dev/v4"
	"go-micro.dev/v4/metadata"
)

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *proto.Request, rsp *proto.Response) error {
	md, _ := metadata.FromContext(ctx)

	// local ip of service
	fmt.Println("local ip is", md["Local"])

	// remote ip of caller
	fmt.Println("remote ip is", md["Remote"])

	rsp.Greeting = "Hello " + req.Name
	return nil
}

func main() {
	service := micro.NewService(
		micro.Name("greeter"),
	)

	service.Init()

	proto.RegisterGreeterHandler(service.Server(), new(Greeter))

	if err := service.Run(); err != nil {
		fmt.Println(err)
	}
}
