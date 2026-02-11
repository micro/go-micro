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
	service := micro.NewService(
		micro.Name("greeter"),
		micro.Version("1.0.0"),
	)

	service.Init()

	// Register handler - documentation extracted automatically from comments!
	handler := service.Server().NewHandler(new(Greeter))
	if err := service.Server().Handle(handler); err != nil {
		log.Fatal(err)
	}

	// Start MCP gateway on port 3000
	go func() {
		log.Println("Starting MCP gateway on :3000")
		if err := mcp.ListenAndServe(":3000", mcp.Options{
			Registry: service.Options().Registry,
		}); err != nil {
			log.Printf("MCP gateway error: %v", err)
		}
	}()

	log.Println("Greeter service starting...")
	log.Println("Service: greeter")
	log.Println("Endpoint: Greeter.SayHello")
	log.Println("MCP Gateway: http://localhost:3000")
	log.Println("")
	log.Println("Test with:")
	log.Println("  curl http://localhost:3000/mcp/tools")
	log.Println("")
	log.Println("Or use with Claude Code:")
	log.Println("  micro mcp serve")

	// Run service
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
