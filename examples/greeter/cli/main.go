package main

import (
	"context"
	"fmt"

	hello "github.com/micro/go-micro/examples/greeter/srv/proto/hello"
	"github.com/micro/go-micro/v2"
)

func main() {
	// create a new service
	service := micro.NewService()

	// parse command line flags
	service.Init()

	// Use the generated client stub
	cl := hello.NewSayService("go.micro.srv.greeter", service.Client())

	// Make request
	rsp, err := cl.Hello(context.Background(), &hello.Request{
		Name: "John",
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(rsp.Msg)
}
