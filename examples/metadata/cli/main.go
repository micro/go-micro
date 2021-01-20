package main

import (
	"fmt"

	hello "github.com/asim/go-micro/examples/v3/greeter/srv/proto/hello"
	"github.com/asim/go-micro/v3"
	"github.com/asim/go-micro/v3/metadata"

	"context"
)

func main() {
	service := micro.NewService()
	service.Init()

	cl := hello.NewSayService("go.micro.srv.greeter", service.Client())

	// Set arbitrary headers in context
	ctx := metadata.NewContext(context.Background(), map[string]string{
		"User": "john",
		"ID":   "1",
	})

	rsp, err := cl.Hello(ctx, &hello.Request{
		Name: "John",
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(rsp.Msg)
}
