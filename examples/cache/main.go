package main

import (
	"github.com/asim/go-micro/examples/v4/cache/handler"
	pb "github.com/asim/go-micro/examples/v4/cache/proto"

	"go-micro.dev/v4"
	log "go-micro.dev/v4/logger"
)

var (
	service = "go.micro.srv.cache"
	version = "latest"
)

func main() {
	// Create service
	srv := micro.NewService(
		micro.Name(service),
		micro.Version(version),
	)
	srv.Init()

	// Register handler
	pb.RegisterCacheHandler(srv.Server(), handler.NewCache())

	// Run service
	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}
