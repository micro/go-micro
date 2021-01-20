package main

import (
	"log"
	"time"

	"context"
	"github.com/asim/go-micro/v3"
)

func main() {
	// cancellation context
	ctx, cancel := context.WithCancel(context.Background())

	// shutdown after 5 seconds
	go func() {
		<-time.After(time.Second * 5)
		log.Println("Shutdown example: shutting down service")
		cancel()
	}()

	// create service
	service := micro.NewService(
		// with our cancellation context
		micro.Context(ctx),
	)

	// init service
	service.Init()

	// run service
	service.Run()
}
