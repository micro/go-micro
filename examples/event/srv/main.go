package main

import (
	"context"

	"github.com/asim/go-micro/v3"
	proto "github.com/asim/go-micro/v3/api/proto"
	"github.com/asim/go-micro/v3/util/log"
)

// All methods of Event will be executed when a message is received
type Event struct{}

// Method can be of any name
func (e *Event) Process(ctx context.Context, event *proto.Event) error {
	log.Logf("Received event %+v\n", event)
	// do something with event
	return nil
}

func main() {
	service := micro.NewService(
		micro.Name("user"),
	)
	service.Init()

	// register subscriber
	micro.RegisterSubscriber("go.micro.evt.user", service.Server(), new(Event))

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
