package main

import (
	"fmt"

	h "github.com/grpc/grpc-common/go/helloworld"
	"github.com/myodc/go-micro/client"
	"golang.org/x/net/context"
)

// run github.com/grpc/grpc-common/go/greeter_server/main.go
func main() {
	req := client.NewRpcRequest("helloworld.Greeter", "helloworld.Greeter/SayHello", &h.HelloRequest{
		Name: "John",
	}, "application/grpc")

	rsp := &h.HelloReply{}
	err := client.CallRemote(context.Background(), "localhost:50051", req, rsp)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(rsp.Message)
}
