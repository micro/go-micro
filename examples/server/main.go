package main

import (
	log "github.com/golang/glog"
	"github.com/micro/go-micro/cmd"
	"github.com/micro/go-micro/examples/server/handler"
	"github.com/micro/go-micro/examples/server/subscriber"
	"github.com/micro/go-micro/server"
	"golang.org/x/net/context"
)

func logWrapper(fn server.HandlerFunc) server.HandlerFunc {
	return func(ctx context.Context, req interface{}, rsp interface{}) error {
		log.Infof("[Log Wrapper] Before serving request")
		err := fn(ctx, req, rsp)
		log.Infof("[Log Wrapper] After serving request")
		return err
	}
}

func main() {
	// optionally setup command line usage
	cmd.Init()

	server.DefaultServer = server.NewServer(
		server.WrapHandler(logWrapper),
	)

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
