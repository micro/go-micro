package main

import (
	"fmt"
	"time"

	"github.com/asim/go-micro/broker"
)

var (
	topic = "go.micro.topic.foo"
)

func pub() {
	tick := time.NewTicker(time.Second)
	i := 0
	for _ = range tick.C {
		msg := fmt.Sprintf("%d: %s", i, time.Now().String())
		fmt.Println("[pub] pubbed message:", msg)
		broker.Publish(topic, []byte(msg))
		i++
	}
}

func sub() {
	_, err := broker.Subscribe(topic, func(msg *broker.Message) {
		fmt.Println("[sub] received message:", string(msg.Data))
	})
	if err != nil {
		fmt.Println(err)
	}
}

func main() {
	broker.Init()
	broker.Connect()

	go pub()
	go sub()

	<-time.After(time.Second * 10)
}
