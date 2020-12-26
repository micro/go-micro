package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/micro/go-micro/v2/web"
)

func helloWorldHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `<html><body><h1>Hello World</h1></body></html>`)
}

func main() {
	service := web.NewService(
		web.Name("go.micro.web.helloworld"),
		web.Icon("https://www.thefishsociety.co.uk/media/image/e3/1b/b0/prawn-ix.jpg"),
	)

	service.HandleFunc("/", helloWorldHandler)

	if err := service.Init(); err != nil {
		log.Fatal(err)
	}

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
