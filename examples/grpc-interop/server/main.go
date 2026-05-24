// server starts a go-micro gRPC service. Any standard gRPC client
// (Go, Python, Java, etc.) can call it — no go-micro SDK required on
// the client side.
package main

import (
	"context"
	"fmt"
	"log"

	micro "go-micro.dev/v5"
	"go-micro.dev/v5/client"
	grpcclient "go-micro.dev/v5/client/grpc"
	"go-micro.dev/v5/server"
	grpcserver "go-micro.dev/v5/server/grpc"

	pb "example/proto"
)

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *pb.HelloRequest, rsp *pb.HelloResponse) error {
	log.Printf("Received request: name=%q", req.Name)
	rsp.Message = "Hello " + req.Name
	return nil
}

func main() {
	addr := ":50051"

	service := micro.New("greeter",
		micro.Server(grpcserver.NewServer(
			server.Name("greeter"),
			server.Address(addr),
		)),
		micro.Client(grpcclient.NewClient(
			client.ContentType("application/grpc+proto"),
		)),
	)

	service.Init()

	pb.RegisterGreeterHandler(service.Server(), new(Greeter))

	fmt.Println("Go-Micro gRPC server listening on", addr)
	fmt.Println()
	fmt.Println("Call with a standard gRPC client:")
	fmt.Println("  go run ../client/main.go")

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
