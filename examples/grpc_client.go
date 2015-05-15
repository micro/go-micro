package main

import (
	"fmt"

	h "github.com/grpc/grpc-common/go/helloworld"
	"github.com/myodc/go-micro/client"
)

// run github.com/grpc/grpc-common/go/greeter_server/main.go
func main() {
	req := client.NewRpcRequest("helloworld.Greeter", "SayHello", &h.HelloRequest{
		Name: "John",
	}, "application/grpc")

	rsp := &h.HelloReply{}
	err := client.CallRemote("localhost:50051", "helloworld.Greeter/SayHello", req, rsp)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(rsp.Message)
}
