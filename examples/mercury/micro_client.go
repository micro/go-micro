package main

import (
	"fmt"

	"github.com/micro/go-micro/client"
	mcodec "github.com/micro/go-plugins/codec/mercury"
	"github.com/micro/go-plugins/selector/mercury"
	"github.com/micro/go-plugins/transport/rabbitmq"
	hello "github.com/micro/micro/examples/greeter/server/proto/hello"

	"golang.org/x/net/context"
)

func main() {
	rabbitmq.DefaultExchange = "b2a"
	rabbitmq.DefaultRabbitURL = "amqp://localhost:5672"

	c := client.NewClient(
		client.Selector(mercury.NewSelector()),
		client.Transport(rabbitmq.NewTransport([]string{})),
		client.Codec("application/x-protobuf", mcodec.NewCodec),
		client.ContentType("application/x-protobuf"),
	)

	req := c.NewRequest("foo", "Say.Hello", &hello.Request{
		Name: "John",
	})

	rsp := &hello.Response{}

	if err := c.Call(context.Background(), req, rsp); err != nil {
		fmt.Println(err)
	}

	fmt.Println(rsp)
}
