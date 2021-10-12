package main

import (
	"fmt"

	"go-micro.dev/v4"
	"go-micro.dev/v4/server"
)

func main() {
	service := micro.NewService()

	service.Server().Init(
		server.Wait(nil),
	)

	if err := service.Run(); err != nil {
		fmt.Println(err)
	}
}
