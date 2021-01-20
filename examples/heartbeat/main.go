package main

import (
	"time"

	"github.com/asim/go-micro/v3"
	"github.com/asim/go-micro/v3/util/log"
)

func main() {
	service := micro.NewService(
		micro.Name("com.example.srv.foo"),
		micro.RegisterTTL(time.Second*30),
		micro.RegisterInterval(time.Second*15),
	)
	service.Init()

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
