package rabbitmq_test

import (
	"context"
	"os"
	"testing"

	micro "go-micro.dev/v4"
	broker "go-micro.dev/v4/broker"
	server "go-micro.dev/v4/server"
	rabbitmq "github.com/asim/go-micro/plugins/broker/rabbitmq/v4"
)

type Example struct{}

func (e *Example) Handler(ctx context.Context, r interface{}) error {
	return nil
}

func TestDurable(t *testing.T) {
	if tr := os.Getenv("TRAVIS"); len(tr) > 0 {
		t.Skip()
	}
	rabbitmq.DefaultRabbitURL = "amqp://rabbitmq:rabbitmq@127.0.0.1:5672"
	brkrSub := broker.NewSubscribeOptions(
		broker.Queue("queue.default"),
		broker.DisableAutoAck(),
		rabbitmq.DurableQueue(),
	)

	b := rabbitmq.NewBroker()
	b.Init()
	if err := b.Connect(); err != nil {
		t.Logf("cant conect to broker, skip: %v", err)
		t.Skip()
	}

	s := server.NewServer(server.Broker(b))

	service := micro.NewService(
		micro.Server(s),
		micro.Broker(b),
	)
	h := &Example{}
	// Register a subscriber
	micro.RegisterSubscriber(
		"topic",
		service.Server(),
		h.Handler,
		server.SubscriberContext(brkrSub.Context),
		server.SubscriberQueue("queue.default"),
	)

	//service.Init()

	if err := service.Run(); err != nil {
		t.Fatal(err)
	}

}
