package main

import (
	"fmt"
	"log"
	"time"

	hello "github.com/asim/go-micro/examples/v3/greeter/srv/proto/hello"
	"github.com/asim/go-micro/v3"
	"github.com/asim/go-micro/v3/metadata"

	"context"
)

type Say struct{}

func (s *Say) Hello(ctx context.Context, req *hello.Request, rsp *hello.Response) error {
	md, ok := metadata.FromContext(ctx)
	if !ok {
		rsp.Msg = "No metadata received"
		return nil
	}
	log.Printf("Received metadata %v\n", md)
	rsp.Msg = fmt.Sprintf("Hello %s thanks for this %v", req.Name, md)
	return nil
}

func main() {
	service := micro.NewService(
		micro.Name("go.micro.srv.greeter"),
		micro.RegisterTTL(time.Second*30),
		micro.RegisterInterval(time.Second*10),
	)

	service.Init()

	hello.RegisterSayHandler(service.Server(), new(Say))

	// Run server
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
