package resource

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v6/broker"
)

// brokerCommand exposes the broker interface: publish, subscribe.
func brokerCommand() *cli.Command {
	return &cli.Command{
		Name:  "broker",
		Usage: "Publish and subscribe to broker topics",
		Description: `Interact with the message broker.

  micro broker publish <topic> <message>   Publish a message to a topic
  micro broker subscribe <topic>           Stream messages from a topic`,
		Subcommands: []*cli.Command{
			{
				Name:      "publish",
				Usage:     "Publish a message to a topic",
				ArgsUsage: "<topic> <message>",
				Action:    brokerPublish,
			},
			{
				Name:      "subscribe",
				Usage:     "Stream messages from a topic",
				ArgsUsage: "<topic>",
				Action:    brokerSubscribe,
			},
		},
	}
}

func brokerPublish(c *cli.Context) error {
	topic := c.Args().Get(0)
	msg := c.Args().Get(1)
	if topic == "" || msg == "" {
		return fail("usage: micro broker publish <topic> <message>")
	}

	b := broker.DefaultBroker
	if err := b.Connect(); err != nil {
		return fail("broker connect: %v", err)
	}

	if err := b.Publish(topic, &broker.Message{Body: []byte(msg)}); err != nil {
		return fail("publish: %v", err)
	}

	fmt.Printf("Published to %q\n", topic)
	return nil
}

func brokerSubscribe(c *cli.Context) error {
	topic := c.Args().First()
	if topic == "" {
		return fail("usage: micro broker subscribe <topic>")
	}

	b := broker.DefaultBroker
	if err := b.Connect(); err != nil {
		return fail("broker connect: %v", err)
	}

	sub, err := b.Subscribe(topic, func(e broker.Event) error {
		fmt.Printf("%s\n", string(e.Message().Body))
		return nil
	})
	if err != nil {
		return fail("subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	fmt.Printf("Subscribed to %q (Ctrl-C to stop)...\n", topic)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	return nil
}
