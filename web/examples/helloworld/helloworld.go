package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/micro/go-micro/web"
)

func helloWorldHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `<html><body><h1>Hello World</h1></body></html>`)
}

func main() {
	service := web.NewService(
		web.Name("helloworld"),
	)

	service.HandleFunc("/", helloWorldHandler)

	if err := service.Init(); err != nil {
		log.Fatal(err)
	}

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
