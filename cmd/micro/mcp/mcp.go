// Package mcp provides the 'micro mcp' command for MCP server management
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/cmd"
	"go-micro.dev/v5/codec/bytes"
	"go-micro.dev/v5/gateway/mcp"
	"go-micro.dev/v5/registry"
)

func init() {
	cmd.Register(&cli.Command{
		Name:  "mcp",
		Usage: "MCP server management",
		Description: `Manage MCP (Model Context Protocol) server for AI agent integration.

Examples:
  # Start MCP server (stdio for Claude Code)
  micro mcp serve

  # Start MCP server with HTTP/SSE
  micro mcp serve --address :3000

  # List available tools
  micro mcp list

  # Test a tool
  micro mcp test users.Users.Get

The 'micro mcp' command exposes your microservices as AI-accessible tools via the
Model Context Protocol (MCP). This enables Claude Code, ChatGPT, and other AI agents
to discover and call your services automatically.

For Claude Code integration, add to your config:
  {
    "mcpServers": {
      "my-services": {
        "command": "micro",
        "args": ["mcp", "serve"]
      }
    }
  }`,
		Subcommands: []*cli.Command{
			{
				Name:  "serve",
				Usage: "Start MCP server",
				Description: `Start an MCP server to expose microservices as AI tools.

By default, uses stdio transport (for Claude Code and local AI tools).
Use --address for HTTP/SSE transport (for web-based agents).

Examples:
  # Stdio transport (for Claude Code)
  micro mcp serve

  # HTTP/SSE transport
  micro mcp serve --address :3000

  # Custom registry
  micro mcp serve --registry consul --registry_address consul:8500`,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "address",
						Usage: "HTTP address to listen on (e.g., :3000). If not set, uses stdio.",
					},
					&cli.StringFlag{
						Name:  "registry",
						Usage: "Registry for service discovery (mdns, consul, etcd)",
						Value: "mdns",
					},
					&cli.StringFlag{
						Name:  "registry_address",
						Usage: "Registry address (e.g., consul:8500)",
					},
				},
				Action: serveAction,
			},
			{
				Name:  "list",
				Usage: "List available tools",
				Description: `List all tools available via MCP.

Each service endpoint is exposed as a tool that AI agents can call.

Example:
  micro mcp list`,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "registry",
						Usage: "Registry for service discovery (mdns, consul, etcd)",
						Value: "mdns",
					},
					&cli.StringFlag{
						Name:  "registry_address",
						Usage: "Registry address",
					},
					&cli.BoolFlag{
						Name:  "json",
						Usage: "Output as JSON",
					},
				},
				Action: listAction,
			},
			{
				Name:  "test",
				Usage: "Test a tool",
				Description: `Test calling a specific tool.

Example:
  micro mcp test users.Users.Get '{"id": "123"}'`,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "registry",
						Usage: "Registry for service discovery",
						Value: "mdns",
					},
					&cli.StringFlag{
						Name:  "registry_address",
						Usage: "Registry address",
					},
				},
				Action: testAction,
			},
		},
	})
}

// serveAction starts the MCP server
func serveAction(ctx *cli.Context) error {
	// Get registry
	reg := registry.DefaultRegistry
	if regName := ctx.String("registry"); regName != "" {
		// TODO: Support other registries (consul, etcd)
		if regName != "mdns" {
			return fmt.Errorf("registry %s not yet supported, use mdns", regName)
		}
	}

	// Create MCP server options
	opts := mcp.Options{
		Registry: reg,
		Address:  ctx.String("address"),
		Context:  context.Background(),
		Logger:   log.Default(),
	}

	// Handle shutdown gracefully
	ctx2, cancel := context.WithCancel(opts.Context)
	opts.Context = ctx2
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Start MCP server
	return mcp.Serve(opts)
}

// listAction lists available tools
func listAction(ctx *cli.Context) error {
	// Get registry
	reg := registry.DefaultRegistry

	// Create temporary MCP server to discover tools
	opts := mcp.Options{
		Registry: reg,
		Context:  context.Background(),
		Logger:   log.New(os.Stderr, "", 0), // Log to stderr so stdout is clean
	}

	// Discover services
	services, err := opts.Registry.ListServices()
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	if ctx.Bool("json") {
		// JSON output
		var tools []map[string]interface{}
		for _, svc := range services {
			fullSvcs, err := opts.Registry.GetService(svc.Name)
			if err != nil || len(fullSvcs) == 0 {
				continue
			}

			for _, ep := range fullSvcs[0].Endpoints {
				tools = append(tools, map[string]interface{}{
					"name":        fmt.Sprintf("%s.%s", svc.Name, ep.Name),
					"service":     svc.Name,
					"endpoint":    ep.Name,
					"description": fmt.Sprintf("Call %s on %s service", ep.Name, svc.Name),
				})
			}
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]interface{}{
			"tools": tools,
			"count": len(tools),
		})
	}

	// Human-readable output
	fmt.Printf("Available MCP Tools:\n\n")
	toolCount := 0
	for _, svc := range services {
		fullSvcs, err := opts.Registry.GetService(svc.Name)
		if err != nil || len(fullSvcs) == 0 {
			continue
		}

		fmt.Printf("Service: %s\n", svc.Name)
		for _, ep := range fullSvcs[0].Endpoints {
			toolName := fmt.Sprintf("%s.%s", svc.Name, ep.Name)
			fmt.Printf("  • %s\n", toolName)
			toolCount++
		}
		fmt.Println()
	}

	fmt.Printf("Total: %d tools\n", toolCount)
	return nil
}

// testAction tests a specific tool
func testAction(ctx *cli.Context) error {
	if ctx.Args().Len() < 1 {
		return fmt.Errorf("usage: micro mcp test <tool-name> [input-json]")
	}

	toolName := ctx.Args().First()
	inputJSON := "{}"
	if ctx.Args().Len() > 1 {
		inputJSON = ctx.Args().Get(1)
	}

	// Get registry
	reg := registry.DefaultRegistry

	// Parse tool name (format: service.endpoint or service.Handler.Method)
	parts := splitToolName(toolName)
	if len(parts) < 2 {
		return fmt.Errorf("invalid tool name format: %s (expected format: service.Endpoint or service.Handler.Method)", toolName)
	}

	serviceName := parts[0]
	var endpointName string
	if len(parts) == 2 {
		endpointName = parts[1]
	} else {
		// For format like greeter.Greeter.SayHello, endpoint is Greeter.SayHello
		endpointName = strings.Join(parts[1:], ".")
	}

	// Verify service exists
	services, err := reg.GetService(serviceName)
	if err != nil || len(services) == 0 {
		return fmt.Errorf("service not found: %s", serviceName)
	}

	// Verify endpoint exists
	endpointFound := false
	for _, ep := range services[0].Endpoints {
		if ep.Name == endpointName {
			endpointFound = true
			break
		}
	}
	if !endpointFound {
		return fmt.Errorf("endpoint not found: %s on service %s", endpointName, serviceName)
	}

	fmt.Printf("Testing tool: %s\n", toolName)
	fmt.Printf("Input: %s\n\n", inputJSON)

	// Parse input JSON
	var input map[string]interface{}
	if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
		return fmt.Errorf("invalid JSON input: %w", err)
	}

	// Create MCP server options for making the call
	opts := mcp.Options{
		Registry: reg,
		Context:  context.Background(),
		Logger:   log.New(os.Stderr, "", 0),
	}

	// Make RPC call using client
	c := client.DefaultClient
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal input: %w", err)
	}

	rpcReq := c.NewRequest(serviceName, endpointName, &bytes.Frame{Data: inputBytes})
	var rsp bytes.Frame

	if err := c.Call(opts.Context, rpcReq, &rsp); err != nil {
		return fmt.Errorf("RPC call failed: %w", err)
	}

	// Parse and display response
	var result interface{}
	if err := json.Unmarshal(rsp.Data, &result); err != nil {
		// If unmarshal fails, display raw data
		fmt.Printf("Result (raw): %s\n", string(rsp.Data))
	} else {
		// Pretty print JSON result
		prettyJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Printf("Result: %v\n", result)
		} else {
			fmt.Printf("Result:\n%s\n", string(prettyJSON))
		}
	}

	return nil
}

// splitToolName splits a tool name like "greeter.Greeter.SayHello" into parts
func splitToolName(name string) []string {
	return strings.Split(name, ".")
}
