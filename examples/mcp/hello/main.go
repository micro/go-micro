// Package main demonstrates a minimal MCP-enabled service.
//
// This is the simplest possible example showing:
// - Automatic documentation extraction from Go comments
// - MCP gateway setup
// - Ready for use with Claude Code
package main

import (
	"context"
	"log"

	"go-micro.dev/v5"
	"go-micro.dev/v5/gateway/mcp"
)

// Greeter service handles greeting operations
type Greeter struct{}

// SayHello greets a person by name. Returns a friendly greeting message.
//
// @example {"name": "Alice"}
func (g *Greeter) SayHello(ctx context.Context, req *HelloRequest, rsp *HelloResponse) error {
	rsp.Message = "Hello " + req.Name + "!"
	return nil
}

// HelloRequest contains the greeting parameters
type HelloRequest struct {
	Name string `json:"name" description:"Person's name to greet"`
}

// HelloResponse contains the greeting result
type HelloResponse struct {
	Message string `json:"message" description:"The greeting message"`
}

func main() {
	// Create service
	service := micro.New("greeter",
		micro.Address(":9090"),
		// Start MCP gateway alongside the service
		mcp.WithMCP(":3000"),
	)

	service.Init()

	// Register handler — docs extracted automatically from comments
	if err := service.Handle(new(Greeter)); err != nil {
		log.Fatal(err)
	}

	log.Println("Greeter service starting...")
	log.Println("Service:     http://localhost:9090")
	log.Println("MCP Gateway: http://localhost:3000")
	log.Println("MCP Tools:   http://localhost:3000/mcp/tools")
	log.Println()
	log.Println("Use with Claude Code:")
	log.Println("  micro mcp serve")

	// Run service
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
