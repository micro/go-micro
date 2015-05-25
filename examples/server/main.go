package main

import (
	log "github.com/golang/glog"
	"github.com/myodc/go-micro/cmd"
	"github.com/myodc/go-micro/examples/server/handler"
	"github.com/myodc/go-micro/server"
)

func main() {
	// optionally setup command line usage
	cmd.Init()

	server.Name = "go.micro.srv.example"

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
