package main

import (
	"fmt"
	"log"
	"net/http"

	hello "github.com/micro/go-micro/examples/greeter/srv/proto/hello"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/web"

	"context"
)

func main() {
	service := web.NewService(
		web.Name("go.micro.web.greeter"),
	)

	service.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			r.ParseForm()

			name := r.Form.Get("name")
			if len(name) == 0 {
				name = "World"
			}

			cl := hello.NewSayService("go.micro.srv.greeter", client.DefaultClient)
			rsp, err := cl.Hello(context.Background(), &hello.Request{
				Name: name,
			})

			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			w.Write([]byte(`<html><body><h1>` + rsp.Msg + `</h1></body></html>`))
			return
		}

		fmt.Fprint(w, `<html><body><h1>Enter Name<h1><form method=post><input name=name type=text /></form></body></html>`)
	})

	if err := service.Init(); err != nil {
		log.Fatal(err)
	}

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
