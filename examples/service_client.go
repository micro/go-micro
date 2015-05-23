package main

import (
	"fmt"

	"github.com/myodc/go-micro/client"
	"github.com/myodc/go-micro/cmd"
	c "github.com/myodc/go-micro/context"
	example "github.com/myodc/go-micro/template/proto/example"
	"golang.org/x/net/context"
)

func main() {
	cmd.Init()
	// Create new request to service go.micro.service.go-template, method Example.Call
	req := client.NewRequest("go.micro.service.template", "Example.Call", &example.Request{
		Name: "John",
	})

	// create context with metadata
	ctx := c.WithMetaData(context.Background(), map[string]string{
		"X-User-Id": "john",
		"X-From-Id": "script",
	})

	rsp := &example.Response{}

	// Call service
	if err := client.Call(ctx, req, rsp); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(rsp.Msg)
}
