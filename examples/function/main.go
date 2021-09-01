package main

import (
	"context"

	proto "github.com/chinahtl/go-micro/examples/v3/function/proto"
	"github.com/chinahtl/go-micro/v3"
)

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *proto.HelloRequest, rsp *proto.HelloResponse) error {
	rsp.Greeting = "Hello " + req.Name
	return nil
}

func main() {
	fnc := micro.NewFunction(
		micro.Name("greeter"),
	)

	fnc.Init()

	fnc.Handle(new(Greeter))

	fnc.Run()
}
