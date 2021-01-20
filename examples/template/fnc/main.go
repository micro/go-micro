package main

import (
	"github.com/micro/go-micro/examples/template/fnc/handler"
	"github.com/micro/go-micro/examples/template/fnc/subscriber"
	"github.com/asim/go-micro/v3"
	"github.com/asim/go-micro/v3/util/log"
)

func main() {
	// New Service
	function := micro.NewFunction(
		micro.Name("go.micro.fnc.template"),
		micro.Version("latest"),
	)

	// Register Handler
	function.Handle(new(handler.Example))

	// Register Struct as Subscriber
	function.Subscribe("go.micro.fnc.template", new(subscriber.Example))

	// Initialise function
	function.Init()

	// Run service
	if err := function.Run(); err != nil {
		log.Fatal(err)
	}
}
