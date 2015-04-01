package main

//go:generate protoc -I ./proto ./proto/service.proto --go_out=plugins=grpc:proto

import (
	"github.com/asim/go-micro/cmd"
	"github.com/asim/go-micro/server"
	log "github.com/golang/glog"
	"google.golang.org/grpc"

	"github.com/asim/go-micro/examples/greeter-service/handlers"

	pb "github.com/asim/go-micro/examples/greeter-service/proto"
)

func main() {
	server.Name = "go.micro.service.greeter"

	// Initialise Server
	cmd.Init()
	server.Init()

	// Register Handlers
	handlers := handlers.New()
	server.Register(server.GRPCHandler(func(s *grpc.Server) {
		pb.RegisterGreeterServer(s, handlers)
	}))

	// Run server
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
