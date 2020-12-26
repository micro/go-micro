package main

import (
	proto "github.com/micro/go-micro/examples/pubsub/srv/proto"
	"github.com/micro/go-micro/v2"
	"github.com/micro/go-micro/v2/metadata"
	"github.com/micro/go-micro/v2/server"
	"github.com/micro/go-micro/v2/util/log"

	"context"
)

// All methods of Sub will be executed when
// a message is received
type Sub struct{}

// Method can be of any name
func (s *Sub) Process(ctx context.Context, event *proto.Event) error {
	md, _ := metadata.FromContext(ctx)
	log.Logf("[pubsub.1] Received event %+v with metadata %+v\n", event, md)
	// do something with event
	return nil
}

// Alternatively a function can be used
func subEv(ctx context.Context, event *proto.Event) error {
	md, _ := metadata.FromContext(ctx)
	log.Logf("[pubsub.2] Received event %+v with metadata %+v\n", event, md)
	// do something with event
	return nil
}

func main() {
	// create a service
	service := micro.NewService(
		micro.Name("go.micro.srv.pubsub"),
	)
	// parse command line
	service.Init()

	// register subscriber
	micro.RegisterSubscriber("example.topic.pubsub.1", service.Server(), new(Sub))

	// register subscriber with queue, each message is delivered to a unique subscriber
	micro.RegisterSubscriber("example.topic.pubsub.2", service.Server(), subEv, server.SubscriberQueue("queue.pubsub"))

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
