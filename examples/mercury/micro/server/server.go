package main

import (
	"flag"
	"github.com/micro/go-micro/server"
	mcodec "github.com/micro/go-plugins/codec/mercury"
	"github.com/micro/go-plugins/transport/rabbitmq"
	hello "github.com/micro/micro/examples/greeter/server/proto/hello"

	"golang.org/x/net/context"
)

type Say struct{}

func (s *Say) Hello(ctx context.Context, req *hello.Request, rsp *hello.Response) error {
	rsp.Msg = "Hey " + req.Name
	return nil
}

func main() {
	flag.Parse()
	rabbitmq.DefaultExchange = "b2a"
	rabbitmq.DefaultRabbitURL = "amqp://localhost:5672"

	s := server.NewServer(
		server.Name("foo"),
		server.Id("foo"),
		server.Address("foo"),
		server.Transport(rabbitmq.NewTransport([]string{})),
		server.Codec("application/x-protobuf", mcodec.NewCodec),
	)
	s.Handle(
		s.NewHandler(&Say{}),
	)

	s.Start()
	select {}
}
