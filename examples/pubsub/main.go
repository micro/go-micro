package main

import (
	"fmt"
	"time"

	log "github.com/golang/glog"
	"github.com/myodc/go-micro/broker"
	"github.com/myodc/go-micro/cmd"
	c "github.com/myodc/go-micro/context"

	"golang.org/x/net/context"
)

var (
	topic = "go.micro.topic.foo"
)

func pub() {
	tick := time.NewTicker(time.Second)
	i := 0
	for _ = range tick.C {
		ctx := c.WithMetadata(context.Background(), map[string]string{
			"id": fmt.Sprintf("%d", i),
		})

		msg := fmt.Sprintf("%d: %s", i, time.Now().String())
		if err := broker.Publish(ctx, topic, []byte(msg)); err != nil {
			log.Errorf("[pub] failed: %v", err)
		} else {
			fmt.Println("[pub] pubbed message:", msg)
		}
		i++
	}
}

func sub() {
	_, err := broker.Subscribe(topic, func(ctx context.Context, msg *broker.Message) {
		md, _ := c.GetMetadata(ctx)
		fmt.Println("[sub] received message:", string(msg.Body), "context", md)
	})
	if err != nil {
		fmt.Println(err)
	}
}

func main() {
	cmd.Init()

	if err := broker.Init(); err != nil {
		log.Fatalf("Broker Init error: %v", err)
	}
	if err := broker.Connect(); err != nil {
		log.Fatalf("Broker Connect error: %v", err)
	}

	go pub()
	go sub()

	<-time.After(time.Second * 10)
}
