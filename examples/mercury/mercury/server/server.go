package main

import (
	"github.com/mondough/mercury"
	"github.com/mondough/mercury/server"
	"github.com/mondough/mercury/service"
	"github.com/mondough/typhon/rabbit"

	hello "github.com/micro/micro/examples/greeter/server/proto/hello"
)

func handler(req mercury.Request) (mercury.Response, error) {
	request := req.Body().(*hello.Request)
	rsp := req.Response(&hello.Response{
		Msg: "Hey " + request.Name,
	})
	return rsp, nil
}

func main() {
	s := service.Init(service.Config{
		Name:      "foo",
		Transport: rabbit.NewTransport(),
	})

	s.Server().AddEndpoints(server.Endpoint{
		Name:     "Say.Hello",
		Handler:  handler,
		Request:  new(hello.Request),
		Response: new(hello.Response),
	})

	s.Run()
}
