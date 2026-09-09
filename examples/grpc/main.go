// server starts a go-micro gRPC service and enables the reflection API, so
// any standard gRPC client (grpcurl, grpc_cli, Postman, Go, Python, Java,
// ...) can discover and call it — no go-micro SDK required on the client.
package main

import (
	"context"
	"fmt"
	"log"

	micro "go-micro.dev/v6"
	"go-micro.dev/v6/server"
	grpcserver "go-micro.dev/v6/server/grpc"

	pb "example/proto"
)

type Say struct{}

func (s *Say) Hello(ctx context.Context, req *pb.Request, rsp *pb.Response) error {
	rsp.Message = "Hello " + req.Name
	return nil
}

func main() {
	service := micro.NewService("helloworld",
		micro.Server(grpcserver.NewServer(
			// The registry name lives on the gRPC server, not on
			// micro.NewService, otherwise it is lost when the default
			// server is swapped out.
			server.Name("helloworld"),
			server.Address(":8080"),
			// Expose the reflection API so grpcurl and friends can
			// list and describe the service.
			grpcserver.Reflection(),
		)),
	)
	service.Init()

	if err := pb.RegisterSayHandler(service.Server(), &Say{}); err != nil {
		log.Fatal(err)
	}

	fmt.Println("go-micro gRPC server listening on :8080 (service: helloworld)")
	fmt.Println("List services: grpcurl -plaintext localhost:8080 list")

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
