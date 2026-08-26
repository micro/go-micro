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
	"log/slog"

	"go-micro.dev/v6"
	"go-micro.dev/v6/gateway/mcp"
	"go-micro.dev/v6/registry/nats"
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
	rnats := nats.NewNatsRegistry()
	service := micro.NewService(
		"greeter",
		micro.Address(":9091"),
		// Start MCP gateway alongside the service
		mcp.WithMCP(":3002"),
		micro.Registry(rnats),
	)

	service.Init()

	// Register handler — docs extracted automatically from comments
	if err := service.Handle(new(Greeter)); err != nil {
		log.Fatal(err)
	}

	slog.Info("started", "service", "greeter", "addr", ":9091", "mcp", "http://localhost:3002/mcp/tools")

	// Run service
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
