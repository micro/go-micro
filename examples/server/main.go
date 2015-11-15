package main

import (
	log "github.com/golang/glog"
	"github.com/piemapping/go-micro/cmd"
	"github.com/piemapping/go-micro/examples/server/handler"
	"github.com/piemapping/go-micro/examples/server/subscriber"
	"github.com/piemapping/go-micro/server"
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
	server.Subscribe(
		server.NewSubscriber(
			"topic.go.micro.srv.example",
			new(subscriber.Example),
		),
	)

	server.Subscribe(
		server.NewSubscriber(
			"topic.go.micro.srv.example",
			subscriber.Handler,
		),
	)

	// Run server
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
