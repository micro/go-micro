package mcp

import (
	"go-micro.dev/v5/service"
)

// WithMCP returns a service option that starts an MCP gateway alongside the
// service, making all registered handlers discoverable as AI agent tools.
// The address parameter specifies where the MCP gateway listens (e.g., ":3000").
//
// Usage:
//
//	import "go-micro.dev/v5/gateway/mcp"
//
//	service := micro.NewService(
//	    micro.Name("users"),
//	    mcp.WithMCP(":3000"),
//	)
func WithMCP(address string) service.Option {
	return func(o *service.Options) {
		o.AfterStart = append(o.AfterStart, func() error {
			go ListenAndServe(address, Options{
				Registry: o.Registry,
			})
			return nil
		})
	}
}
