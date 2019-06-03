package main

import (
	"context"
	"log"

	"github.com/micro/go-micro"
	"github.com/micro/go-micro/service/grpc"
	hello "github.com/micro/go-micro/service/grpc/examples/greeter/server/proto/hello"
)

type Say struct{}

func (s *Say) Hello(ctx context.Context, req *hello.Request, rsp *hello.Response) error {
	rsp.Msg = "Hello " + req.Name
	return nil
}

func main() {
	fn := grpc.NewFunction(
		micro.Name("go.micro.fnc.greeter"),
	)

	fn.Init()

	fn.Handle(new(Say))

	if err := fn.Run(); err != nil {
		log.Fatal(err)
	}
}
