package rabbitmq_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"go-micro.dev/v5/logger"

	micro "go-micro.dev/v5"
	broker "go-micro.dev/v5/broker"
	rabbitmq "go-micro.dev/v5/broker/rabbitmq"
	server "go-micro.dev/v5/server"
)

type Example struct{}

func init() {
	rabbitmq.DefaultRabbitURL = "amqp://rabbitmq:rabbitmq@127.0.0.1:5672"
}

type TestEvent struct {
	Name string    `json:"name"`
	Age  int       `json:"age"`
	Time time.Time `json:"time"`
}

func (e *Example) Handler(ctx context.Context, r interface{}) error {
	return nil
}

func TestDurable(t *testing.T) {
	if tr := os.Getenv("TRAVIS"); len(tr) > 0 {
		t.Skip()
	}
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

	// service.Init()

	if err := service.Run(); err != nil {
		t.Fatal(err)
	}
}

func TestWithoutExchange(t *testing.T) {

	b := rabbitmq.NewBroker(rabbitmq.WithoutExchange())
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
	brkrSub := broker.NewSubscribeOptions(
		broker.Queue("direct.queue"),
		broker.DisableAutoAck(),
		rabbitmq.DurableQueue(),
	)
	// Register a subscriber
	err := micro.RegisterSubscriber(
		"direct.queue",
		service.Server(),
		func(ctx context.Context, evt *TestEvent) error {
			logger.Logf(logger.InfoLevel, "receive event: %+v", evt)
			return nil
		},
		server.SubscriberContext(brkrSub.Context),
		server.SubscriberQueue("direct.queue"),
	)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(5 * time.Second)
		logger.Logf(logger.InfoLevel, "pub event")
		jsonData, _ := json.Marshal(&TestEvent{
			Name: "test",
			Age:  16,
		})
		err := b.Publish("direct.queue", &broker.Message{
			Body: jsonData,
		},
			rabbitmq.DeliveryMode(2),
			rabbitmq.ContentType("application/json"))
		if err != nil {
			t.Fatal(err)
		}
	}()

	// service.Init()

	if err := service.Run(); err != nil {
		t.Fatal(err)
	}
}

func TestFanoutExchange(t *testing.T) {
	b := rabbitmq.NewBroker(rabbitmq.ExchangeType(rabbitmq.ExchangeTypeFanout), rabbitmq.ExchangeName("fanout.test"))
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
	brkrSub := broker.NewSubscribeOptions(
		broker.Queue("fanout.queue"),
		broker.DisableAutoAck(),
		rabbitmq.DurableQueue(),
	)
	// Register a subscriber
	err := micro.RegisterSubscriber(
		"fanout.queue",
		service.Server(),
		func(ctx context.Context, evt *TestEvent) error {
			logger.Logf(logger.InfoLevel, "receive event: %+v", evt)
			return nil
		},
		server.SubscriberContext(brkrSub.Context),
		server.SubscriberQueue("fanout.queue"),
	)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(5 * time.Second)
		logger.Logf(logger.InfoLevel, "pub event")
		jsonData, _ := json.Marshal(&TestEvent{
			Name: "test",
			Age:  16,
		})
		err := b.Publish("fanout.queue", &broker.Message{
			Body: jsonData,
		},
			rabbitmq.DeliveryMode(2),
			rabbitmq.ContentType("application/json"))
		if err != nil {
			t.Fatal(err)
		}
	}()

	// service.Init()

	if err := service.Run(); err != nil {
		t.Fatal(err)
	}
}

func TestDirectExchange(t *testing.T) {
	b := rabbitmq.NewBroker(rabbitmq.ExchangeType(rabbitmq.ExchangeTypeDirect), rabbitmq.ExchangeName("direct.test"))
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
	brkrSub := broker.NewSubscribeOptions(
		broker.Queue("direct.exchange.queue"),
		broker.DisableAutoAck(),
		rabbitmq.DurableQueue(),
	)
	// Register a subscriber
	err := micro.RegisterSubscriber(
		"direct.exchange.queue",
		service.Server(),
		func(ctx context.Context, evt *TestEvent) error {
			logger.Logf(logger.InfoLevel, "receive event: %+v", evt)
			return nil
		},
		server.SubscriberContext(brkrSub.Context),
		server.SubscriberQueue("direct.exchange.queue"),
	)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(5 * time.Second)
		logger.Logf(logger.InfoLevel, "pub event")
		jsonData, _ := json.Marshal(&TestEvent{
			Name: "test",
			Age:  16,
		})
		err := b.Publish("direct.exchange.queue", &broker.Message{
			Body: jsonData,
		},
			rabbitmq.DeliveryMode(2),
			rabbitmq.ContentType("application/json"))
		if err != nil {
			t.Fatal(err)
		}
	}()

	// service.Init()

	if err := service.Run(); err != nil {
		t.Fatal(err)
	}
}

func TestTopicExchange(t *testing.T) {
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
	brkrSub := broker.NewSubscribeOptions(
		broker.Queue("topic.exchange.queue"),
		broker.DisableAutoAck(),
		rabbitmq.DurableQueue(),
	)
	// Register a subscriber
	err := micro.RegisterSubscriber(
		"my-test-topic",
		service.Server(),
		func(ctx context.Context, evt *TestEvent) error {
			logger.Logf(logger.InfoLevel, "receive event: %+v", evt)
			return nil
		},
		server.SubscriberContext(brkrSub.Context),
		server.SubscriberQueue("topic.exchange.queue"),
	)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(5 * time.Second)
		logger.Logf(logger.InfoLevel, "pub event")
		jsonData, _ := json.Marshal(&TestEvent{
			Name: "test",
			Age:  16,
		})
		err := b.Publish("my-test-topic", &broker.Message{
			Body: jsonData,
		},
			rabbitmq.DeliveryMode(2),
			rabbitmq.ContentType("application/json"))
		if err != nil {
			t.Fatal(err)
		}
	}()

	// service.Init()

	if err := service.Run(); err != nil {
		t.Fatal(err)
	}
}
