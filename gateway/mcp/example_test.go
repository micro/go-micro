package mcp

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"go-micro.dev/v5"
	"go-micro.dev/v5/registry"
)

// Example_inlineGateway shows how to add MCP gateway to an existing service
func Example_inlineGateway() {
	service := micro.NewService(micro.Name("myservice"))
	service.Init()

	// Add MCP gateway alongside your service
	go func() {
		if err := Serve(Options{
			Registry: service.Options().Registry,
			Address:  ":3000",
		}); err != nil {
			log.Fatal(err)
		}
	}()

	// Run your service normally
	service.Run()
}

// Example_standaloneGateway shows how to run MCP gateway as a separate service
func Example_standaloneGateway() {
	// Standalone MCP gateway
	// Discovers all services via registry
	if err := ListenAndServe(":3000", Options{
		Registry: registry.NewMDNSRegistry(),
	}); err != nil {
		log.Fatal(err)
	}
}

// Example_withAuthentication shows how to add authentication
func Example_withAuthentication() {
	service := micro.NewService(micro.Name("myservice"))
	service.Init()

	go func() {
		if err := Serve(Options{
			Registry: service.Options().Registry,
			Address:  ":3000",
			AuthFunc: func(r *http.Request) error {
				token := r.Header.Get("Authorization")
				if token == "" {
					return fmt.Errorf("missing authorization header")
				}
				// Validate token here
				return nil
			},
		}); err != nil {
			log.Fatal(err)
		}
	}()

	service.Run()
}

// Example_customContext shows how to use a custom context for graceful shutdown
func Example_customContext() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	service := micro.NewService(micro.Name("myservice"))
	service.Init()

	go func() {
		if err := Serve(Options{
			Registry: service.Options().Registry,
			Address:  ":3000",
			Context:  ctx,
		}); err != nil {
			log.Fatal(err)
		}
	}()

	service.Run()
	// cancel() will stop the MCP gateway
}
