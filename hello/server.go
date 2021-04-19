package main

import (
	"github.com/micro/go-micro/v2"
	grpcclient "github.com/micro/go-micro/v2/client/grpc"
	log "github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/registry/etcd"
	grpcserver "github.com/micro/go-micro/v2/server/grpc"
	"hello/handler"
	"hello/proto/hello"
)

func main() {
	// New Service
	service := micro.NewService(
		micro.Client(grpcclient.NewClient()),
		micro.Server(grpcserver.NewServer()),
		micro.Registry(etcd.NewRegistry()),
		//micro.Address("127.0.0.1:10086"),
		micro.Name("go.micro.service.hello"),
		micro.Version("latest"),
	)

	// Initialise service
	service.Init()

	// Register Handler
	if err := hello.RegisterHelloHandler(service.Server(), new(handler.Hello)); err != nil {
		panic(err)
	}

	// Run service
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
