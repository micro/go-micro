package mcp

import (
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/service"
)

// WithMCP returns a service option that starts an MCP gateway alongside the
// service, making all registered handlers discoverable as AI agent tools.
// The address parameter specifies where the MCP gateway listens (e.g., ":3000").
//
// Usage:
//
//	import "go-micro.dev/v6/gateway/mcp"
//
//	service := micro.NewService("users",
//	    mcp.WithMCP(":3000"),
//	)
func WithMCP(address string) service.Option {
	return func(o *service.Options) {
		// Build the client from the service's configured registry and
		// transport (so MCP tool calls route over the same NATS transport the
		// service uses, instead of client.DefaultClient's http transport).
		c := client.NewClient(
			client.Registry(o.Registry),
			client.Transport(o.Transport),
		)
		o.AfterStart = append(o.AfterStart, func() error {
			go func() {
				_ = ListenAndServe(address, Options{
					Registry: o.Registry,
					Client:   c,
				})
			}()
			return nil
		})
	}
}
