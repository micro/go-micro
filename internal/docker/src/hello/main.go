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
	"os"
	"strings"

	natslib "github.com/nats-io/nats.go"
	"go-micro.dev/v6"
	"go-micro.dev/v6/broker"
	natsbroker "go-micro.dev/v6/broker/nats"
	"go-micro.dev/v6/gateway/mcp"
	_ "go-micro.dev/v6/otel"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/registry/nats"
	ntx "go-micro.dev/v6/transport/nats"
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
	regAddr := strings.Split(os.Getenv("MICRO_REGISTRY_ADDRESS"), ",")
	brokAddr := strings.Split(os.Getenv("MICRO_BROKER_ADDRESS"), ",")
	natsAddr := strings.Split(os.Getenv("MICRO_TRANSPORT_ADDRESS"), ",")
	mcpAddr := os.Getenv("MICRO_MCP_ADDRESS")
	// Create service
	rnats := nats.NewNatsRegistry(registry.Addrs(regAddr...))
	bnats := natsbroker.NewNatsBroker(broker.Addrs(brokAddr...))
	opts := []micro.Option{
		micro.Registry(rnats),
		micro.Broker(bnats),
		micro.Transport(ntx.NewTransport(ntx.Options(natslib.Options{Servers: natsAddr}))),
	}
	if mcpAddr != "" {
		opts = append(opts, mcp.WithMCP(mcpAddr))
	}
	service := micro.NewService("greeter", opts...)

	service.Init()

	// Register handler — docs extracted automatically from comments
	if err := service.Handle(new(Greeter)); err != nil {
		log.Fatal(err)
	}

	slog.Info("started", "service", "greeter", "mcp", mcpAddr)

	// Run service
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
