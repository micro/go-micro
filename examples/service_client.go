package main

import (
	"fmt"

	"github.com/myodc/go-micro/client"
	example "github.com/myodc/go-micro/template/proto/example"
)

func main() {
	// Create new request to service go.micro.service.go-template, method Example.Call
	req := client.NewRequest("go.micro.service.template", "Example.Call", &example.Request{
		Name: "John",
	})

	// Set arbitrary headers
	req.Headers().Set("X-User-Id", "john")
	req.Headers().Set("X-From-Id", "script")

	rsp := &example.Response{}

	// Call service
	if err := client.Call(req, rsp); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(rsp.Msg)
}
