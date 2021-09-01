package main

import (
	"log"

	"github.com/chinahtl/go-micro/examples/v3/server/handler"
	"github.com/chinahtl/go-micro/examples/v3/server/subscriber"
	"github.com/chinahtl/go-micro/v3/cmd"
	"github.com/chinahtl/go-micro/v3/server"
)

func main() {
	// optionally setup command line usage
	cmd.Init()

	// Initialise Server
	server.Init(
		server.Name("go.micro.srv.example"),
	)

	// Register Handlers
	server.Handle(
		server.NewHandler(
			new(handler.Example),
		),
	)

	// Register Subscribers
	if err := server.Subscribe(
		server.NewSubscriber(
			"topic.example",
			new(subscriber.Example),
		),
	); err != nil {
		log.Fatal(err)
	}

	if err := server.Subscribe(
		server.NewSubscriber(
			"topic.example",
			subscriber.Handler,
		),
	); err != nil {
		log.Fatal(err)
	}

	// Run server
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
