package main

import (
	"github.com/asim/go-micro/cmd"
	"github.com/asim/go-micro/server"
	"github.com/asim/go-micro/template/handler"
	log "github.com/golang/glog"
)

func main() {
	// optionally setup command line usage
	cmd.Init()

	server.Name = "go.micro.service.template"

	// Initialise Server
	server.Init()

	// Register Handlers
	server.Register(
		server.NewReceiver(
			new(handler.Example),
		),
	)

	// Run server
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
