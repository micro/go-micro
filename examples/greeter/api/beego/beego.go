package main

import (
	"context"
	"log"

	"github.com/astaxie/beego"
	bctx "github.com/astaxie/beego/context"

	hello "github.com/micro/go-micro/examples/greeter/srv/proto/hello"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/web"
)

type Say struct{}

var (
	cl hello.SayService
)

func (s *Say) Anything(ctx *bctx.Context) {
	log.Print("Received Say.Anything API request")
	ctx.Output.JSON(map[string]string{
		"message": "Hi, this is the Greeter API",
	}, false, true)
}

func (s *Say) Hello(ctx *bctx.Context) {
	log.Print("Received Say.Hello API request")

	name := ctx.Input.Param(":name")

	response, err := cl.Hello(context.TODO(), &hello.Request{
		Name: name,
	})

	if err != nil {
		ctx.Output.SetStatus(500)
		ctx.Output.JSON(err, false, true)
	}

	ctx.Output.JSON(response, false, true)
}

func main() {
	// Create service
	service := web.NewService(
		web.Name("go.micro.api.greeter"),
	)

	service.Init()

	// Setup Greeter Server Client
	cl = hello.NewSayService("go.micro.srv.greeter", client.DefaultClient)

	// Create RESTful handler
	say := new(Say)
	beego.Get("/greeter", say.Anything)
	beego.Get("/greeter/:name", say.Hello)

	// Register Handler
	service.Handle("/", beego.BeeApp.Handlers)

	// Run server
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
