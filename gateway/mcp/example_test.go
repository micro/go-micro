package mcp

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"go-micro.dev/v5"
	"go-micro.dev/v5/auth/jwt"
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

// Example_withScopesAndTracing shows how to add per-tool scopes, tracing, rate
// limiting and audit logging to the MCP gateway. Services register scope
// requirements via endpoint metadata ("scopes" key, comma-separated).
func Example_withScopesAndTracing() {
	service := micro.NewService(micro.Name("blog"))
	service.Init()

	// Use JWT auth provider
	authProvider := jwt.NewAuth()

	go func() {
		if err := Serve(Options{
			Registry: service.Options().Registry,
			Address:  ":3000",

			// Auth inspects Bearer tokens and enforces per-tool scopes
			Auth: authProvider,

			// Rate limit all tools to 10 req/s with burst of 20
			RateLimit: &RateLimitConfig{
				RequestsPerSecond: 10,
				Burst:             20,
			},

			// Audit every tool call for compliance
			AuditFunc: func(r AuditRecord) {
				log.Printf("[audit] trace=%s tool=%s account=%s allowed=%v reason=%s",
					r.TraceID, r.Tool, r.AccountID, r.Allowed, r.DeniedReason)
			},
		}); err != nil {
			log.Fatal(err)
		}
	}()

	service.Run()
}
