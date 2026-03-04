package template

var (
	MainSRV = `package main

import (
	"{{.Dir}}/handler"
	pb "{{.Dir}}/proto"

	"go-micro.dev/v5"
	"go-micro.dev/v5/gateway/mcp"
)

func main() {
	// Create service
	service := micro.New("{{lower .Alias}}",
		mcp.WithMCP(":3001"),
	)

	// Initialize service
	service.Init()

	// Register handler
	pb.Register{{title .Alias}}Handler(service.Server(), handler.New())

	// Run service
	service.Run()
}
`

	MainSRVNoMCP = `package main

import (
	"{{.Dir}}/handler"
	pb "{{.Dir}}/proto"

	"go-micro.dev/v5"
)

func main() {
	// Create service
	service := micro.New("{{lower .Alias}}")

	// Initialize service
	service.Init()

	// Register handler
	pb.Register{{title .Alias}}Handler(service.Server(), handler.New())

	// Run service
	service.Run()
}
`
)
