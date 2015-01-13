package main

import (
	"fmt"

	"code.google.com/p/goprotobuf/proto"
	"github.com/asim/go-micro/client"
	example "github.com/asim/go-micro/template/proto/example"
)

func main() {
	// Create new request to service go.micro.service.go-template, method Example.Call
	req := client.NewRequest("go.micro.service.template", "Example.Call", &example.Request{
		Name: proto.String("John"),
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

	fmt.Println(rsp.GetMsg())
}
