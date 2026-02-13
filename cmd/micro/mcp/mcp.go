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
	"time"

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
			{
				Name:  "docs",
				Usage: "Generate MCP documentation",
				Description: `Generate documentation for all available MCP tools.

The documentation includes tool names, descriptions, parameters, and examples
extracted from service metadata and Go comments.

Examples:
  # Generate markdown documentation
  micro mcp docs

  # Generate JSON documentation
  micro mcp docs --format json

  # Save to file
  micro mcp docs --output mcp-tools.md`,
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
					&cli.StringFlag{
						Name:  "format",
						Usage: "Output format (markdown, json)",
						Value: "markdown",
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output file (default: stdout)",
					},
				},
				Action: docsAction,
			},
			{
				Name:  "export",
				Usage: "Export tools to different formats",
				Description: `Export MCP tools to various agent framework formats.

Supported formats:
  - langchain: LangChain tool definitions (Python)
  - openapi: OpenAPI 3.0 specification
  - json: Raw JSON tool definitions

Examples:
  # Export to LangChain format
  micro mcp export langchain

  # Export to OpenAPI
  micro mcp export openapi --output openapi.yaml

  # Export raw JSON
  micro mcp export json`,
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
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output file (default: stdout)",
					},
				},
				Action: exportAction,
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

	// Validate input JSON
	var inputData map[string]interface{}
	if err := json.Unmarshal([]byte(inputJSON), &inputData); err != nil {
		return fmt.Errorf("invalid JSON input: %w", err)
	}

	// Get registry
	reg := registry.DefaultRegistry
	if regName := ctx.String("registry"); regName != "" {
		if regName != "mdns" {
			return fmt.Errorf("registry %s not yet supported, use mdns", regName)
		}
	}

	// Create MCP options
	opts := mcp.Options{
		Registry: reg,
		Context:  context.Background(),
		Logger:   log.New(os.Stderr, "", 0),
	}

	// Parse tool name (format: "service.endpoint" or "service.Handler.Method")
	parts := parseTool(toolName)
	if len(parts) < 2 {
		return fmt.Errorf("invalid tool name format. Expected: service.endpoint or service.Handler.Method")
	}

	serviceName := parts[0]
	endpointName := parts[1]
	
	// If tool name has 3 parts, combine last two for endpoint (e.g., Handler.Method)
	if len(parts) == 3 {
		endpointName = parts[1] + "." + parts[2]
	}

	// Discover the tool from registry
	services, err := opts.Registry.GetService(serviceName)
	if err != nil || len(services) == 0 {
		return fmt.Errorf("service %s not found: %w", serviceName, err)
	}

	// Find the endpoint
	var endpoint *registry.Endpoint
	for _, ep := range services[0].Endpoints {
		if ep.Name == endpointName {
			endpoint = ep
			break
		}
	}

	if endpoint == nil {
		return fmt.Errorf("endpoint %s not found in service %s", endpointName, serviceName)
	}

	// Display test info
	fmt.Printf("Testing tool: %s\n", toolName)
	fmt.Printf("Service: %s\n", serviceName)
	fmt.Printf("Endpoint: %s\n", endpointName)
	fmt.Printf("Input: %s\n\n", inputJSON)

	// Convert input to JSON bytes for RPC call
	inputBytes, err := json.Marshal(inputData)
	if err != nil {
		return fmt.Errorf("failed to marshal input: %w", err)
	}

	// Make RPC call using bytes codec
	c := opts.Client
	if c == nil {
		c = client.DefaultClient
	}
	
	// Create request with bytes frame
	req := c.NewRequest(serviceName, endpointName, &bytes.Frame{Data: inputBytes})
	
	// Make the call
	var rsp bytes.Frame
	if err := c.Call(opts.Context, req, &rsp); err != nil {
		fmt.Printf("❌ Call failed: %v\n", err)
		return err
	}

	// Parse and display response
	fmt.Println("✅ Call successful!")
	fmt.Println("\nResponse:")
	
	// Try to pretty-print JSON response
	var result interface{}
	if err := json.Unmarshal(rsp.Data, &result); err == nil {
		prettyJSON, err := json.MarshalIndent(result, "", "  ")
		if err == nil {
			fmt.Println(string(prettyJSON))
		} else {
			fmt.Println(string(rsp.Data))
		}
	} else {
		// Not JSON, print raw
		fmt.Println(string(rsp.Data))
	}

	return nil
}

// parseTool splits a tool name into service and endpoint parts
func parseTool(toolName string) []string {
	return strings.Split(toolName, ".")
}

// docsAction generates documentation for MCP tools
func docsAction(ctx *cli.Context) error {
	// Get registry
	reg := registry.DefaultRegistry
	
	// Create temporary MCP server to discover tools
	opts := mcp.Options{
		Registry: reg,
		Context:  context.Background(),
		Logger:   log.New(os.Stderr, "", 0),
	}

	// Discover services
	services, err := opts.Registry.ListServices()
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	format := ctx.String("format")
	outputFile := ctx.String("output")

	// Prepare output writer
	writer := os.Stdout
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		writer = f
	}

	// Collect all tools with metadata
	type ToolDoc struct {
		Name        string                 `json:"name"`
		Service     string                 `json:"service"`
		Endpoint    string                 `json:"endpoint"`
		Description string                 `json:"description"`
		Example     string                 `json:"example,omitempty"`
		Scopes      []string               `json:"scopes,omitempty"`
		Metadata    map[string]string      `json:"metadata,omitempty"`
	}
	
	var tools []ToolDoc
	for _, svc := range services {
		fullSvcs, err := opts.Registry.GetService(svc.Name)
		if err != nil || len(fullSvcs) == 0 {
			continue
		}

		for _, ep := range fullSvcs[0].Endpoints {
			toolDoc := ToolDoc{
				Name:        fmt.Sprintf("%s.%s", svc.Name, ep.Name),
				Service:     svc.Name,
				Endpoint:    ep.Name,
				Description: fmt.Sprintf("Call %s on %s service", ep.Name, svc.Name),
				Metadata:    ep.Metadata,
			}
			
			// Extract description from metadata if available
			if desc, ok := ep.Metadata["description"]; ok {
				toolDoc.Description = desc
			}
			
			// Extract example from metadata if available
			if example, ok := ep.Metadata["example"]; ok {
				toolDoc.Example = example
			}
			
			// Extract scopes from metadata if available
			if scopesStr, ok := ep.Metadata["scopes"]; ok && scopesStr != "" {
				toolDoc.Scopes = strings.Split(scopesStr, ",")
			}
			
			tools = append(tools, toolDoc)
		}
	}

	// Generate output based on format
	switch format {
	case "json":
		enc := json.NewEncoder(writer)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]interface{}{
			"tools": tools,
			"count": len(tools),
		})
		
	case "markdown":
		fmt.Fprintf(writer, "# MCP Tools Documentation\n\n")
		fmt.Fprintf(writer, "Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Fprintf(writer, "Total Tools: %d\n\n", len(tools))

		
		// Group by service
		serviceMap := make(map[string][]ToolDoc)
		for _, tool := range tools {
			serviceMap[tool.Service] = append(serviceMap[tool.Service], tool)
		}
		
		for service, serviceTools := range serviceMap {
			fmt.Fprintf(writer, "## Service: %s\n\n", service)
			
			for _, tool := range serviceTools {
				fmt.Fprintf(writer, "### %s\n\n", tool.Name)
				fmt.Fprintf(writer, "**Description:** %s\n\n", tool.Description)
				
				if len(tool.Scopes) > 0 {
					fmt.Fprintf(writer, "**Required Scopes:** %s\n\n", strings.Join(tool.Scopes, ", "))
				}
				
				if tool.Example != "" {
					fmt.Fprintf(writer, "**Example Input:**\n```json\n%s\n```\n\n", tool.Example)
				}
			}
		}
		
		return nil
		
	default:
		return fmt.Errorf("unsupported format: %s (supported: markdown, json)", format)
	}
}

// exportAction exports tools to different formats
func exportAction(ctx *cli.Context) error {
	if ctx.Args().Len() < 1 {
		return fmt.Errorf("usage: micro mcp export <format>\nSupported formats: langchain, openapi, json")
	}

	exportFormat := ctx.Args().First()
	
	// Get registry
	reg := registry.DefaultRegistry
	
	// Create temporary MCP server to discover tools
	opts := mcp.Options{
		Registry: reg,
		Context:  context.Background(),
		Logger:   log.New(os.Stderr, "", 0),
	}

	// Discover services
	services, err := opts.Registry.ListServices()
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	outputFile := ctx.String("output")

	// Prepare output writer
	writer := os.Stdout
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		writer = f
	}

	switch exportFormat {
	case "langchain":
		return exportLangChain(writer, services, opts)
	case "openapi":
		return exportOpenAPI(writer, services, opts)
	case "json":
		return exportJSON(writer, services, opts)
	default:
		return fmt.Errorf("unsupported export format: %s\nSupported: langchain, openapi, json", exportFormat)
	}
}

// exportLangChain exports tools in LangChain format (Python)
func exportLangChain(writer *os.File, services []*registry.Service, opts mcp.Options) error {
	fmt.Fprintf(writer, "# LangChain Tools for Go Micro Services\n")
	fmt.Fprintf(writer, "# Auto-generated from MCP service discovery\n\n")
	fmt.Fprintf(writer, "from langchain.tools import Tool\n")
	fmt.Fprintf(writer, "import requests\nimport json\n\n")
	fmt.Fprintf(writer, "# Configure your MCP gateway endpoint\n")
	fmt.Fprintf(writer, "MCP_GATEWAY_URL = 'http://localhost:3000/mcp'\n\n")
	
	fmt.Fprintf(writer, "def call_mcp_tool(tool_name, arguments):\n")
	fmt.Fprintf(writer, "    \"\"\"Call an MCP tool via HTTP gateway\"\"\"\n")
	fmt.Fprintf(writer, "    response = requests.post(\n")
	fmt.Fprintf(writer, "        f'{MCP_GATEWAY_URL}/call',\n")
	fmt.Fprintf(writer, "        json={'name': tool_name, 'arguments': arguments}\n")
	fmt.Fprintf(writer, "    )\n")
	fmt.Fprintf(writer, "    response.raise_for_status()\n")
	fmt.Fprintf(writer, "    return response.json()\n\n")
	
	fmt.Fprintf(writer, "# Define tools\n")
	fmt.Fprintf(writer, "tools = []\n\n")
	
	for _, svc := range services {
		fullSvcs, err := opts.Registry.GetService(svc.Name)
		if err != nil || len(fullSvcs) == 0 {
			continue
		}

		for _, ep := range fullSvcs[0].Endpoints {
			toolName := fmt.Sprintf("%s.%s", svc.Name, ep.Name)
			description := fmt.Sprintf("Call %s on %s service", ep.Name, svc.Name)
			
			if desc, ok := ep.Metadata["description"]; ok {
				description = desc
			}
			
			// Generate Python function name (replace dots with underscores)
			funcName := strings.ReplaceAll(toolName, ".", "_")
			
			fmt.Fprintf(writer, "def %s(arguments: str) -> str:\n", funcName)
			fmt.Fprintf(writer, "    \"\"\"% s\"\"\"\n", description)
			fmt.Fprintf(writer, "    args = json.loads(arguments) if isinstance(arguments, str) else arguments\n")
			fmt.Fprintf(writer, "    return json.dumps(call_mcp_tool('%s', args))\n\n", toolName)
			
			fmt.Fprintf(writer, "tools.append(Tool(\n")
			fmt.Fprintf(writer, "    name='%s',\n", toolName)
			fmt.Fprintf(writer, "    func=%s,\n", funcName)
			fmt.Fprintf(writer, "    description='%s'\n", strings.ReplaceAll(description, "'", "\\'"))
			fmt.Fprintf(writer, "))\n\n")
		}
	}
	
	fmt.Fprintf(writer, "# Example usage:\n")
	fmt.Fprintf(writer, "# from langchain.agents import initialize_agent, AgentType\n")
	fmt.Fprintf(writer, "# from langchain.llms import OpenAI\n")
	fmt.Fprintf(writer, "#\n")
	fmt.Fprintf(writer, "# llm = OpenAI(temperature=0)\n")
	fmt.Fprintf(writer, "# agent = initialize_agent(tools, llm, agent=AgentType.ZERO_SHOT_REACT_DESCRIPTION)\n")
	fmt.Fprintf(writer, "# agent.run('Your query here')\n")
	
	return nil
}

// exportOpenAPI exports tools in OpenAPI 3.0 format
func exportOpenAPI(writer *os.File, services []*registry.Service, opts mcp.Options) error {
	spec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       "Go Micro MCP Services",
			"description": "Auto-generated OpenAPI spec from MCP service discovery",
			"version":     "1.0.0",
		},
		"servers": []map[string]interface{}{
			{
				"url":         "http://localhost:3000",
				"description": "MCP Gateway",
			},
		},
		"paths": make(map[string]interface{}),
	}
	
	paths := spec["paths"].(map[string]interface{})
	
	for _, svc := range services {
		fullSvcs, err := opts.Registry.GetService(svc.Name)
		if err != nil || len(fullSvcs) == 0 {
			continue
		}

		for _, ep := range fullSvcs[0].Endpoints {
			toolName := fmt.Sprintf("%s.%s", svc.Name, ep.Name)
			path := fmt.Sprintf("/mcp/call/%s", strings.ReplaceAll(toolName, ".", "/"))
			
			description := fmt.Sprintf("Call %s on %s service", ep.Name, svc.Name)
			if desc, ok := ep.Metadata["description"]; ok {
				description = desc
			}
			
			operation := map[string]interface{}{
				"summary":     toolName,
				"description": description,
				"operationId": strings.ReplaceAll(toolName, ".", "_"),
				"requestBody": map[string]interface{}{
					"required": true,
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"type": "object",
							},
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Successful response",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
								},
							},
						},
					},
				},
			}
			
			// Add scope security if available
			if scopesStr, ok := ep.Metadata["scopes"]; ok && scopesStr != "" {
				operation["security"] = []map[string]interface{}{
					{
						"bearerAuth": strings.Split(scopesStr, ","),
					},
				}
			}
			
			paths[path] = map[string]interface{}{
				"post": operation,
			}
		}
	}
	
	// Add security schemes
	spec["components"] = map[string]interface{}{
		"securitySchemes": map[string]interface{}{
			"bearerAuth": map[string]interface{}{
				"type":   "http",
				"scheme": "bearer",
			},
		},
	}
	
	enc := json.NewEncoder(writer)
	enc.SetIndent("", "  ")
	return enc.Encode(spec)
}

// exportJSON exports raw tool definitions as JSON
func exportJSON(writer *os.File, services []*registry.Service, opts mcp.Options) error {
	var tools []map[string]interface{}
	
	for _, svc := range services {
		fullSvcs, err := opts.Registry.GetService(svc.Name)
		if err != nil || len(fullSvcs) == 0 {
			continue
		}

		for _, ep := range fullSvcs[0].Endpoints {
			tool := map[string]interface{}{
				"name":     fmt.Sprintf("%s.%s", svc.Name, ep.Name),
				"service":  svc.Name,
				"endpoint": ep.Name,
				"metadata": ep.Metadata,
			}
			
			if desc, ok := ep.Metadata["description"]; ok {
				tool["description"] = desc
			}
			
			if example, ok := ep.Metadata["example"]; ok {
				tool["example"] = example
			}
			
			if scopesStr, ok := ep.Metadata["scopes"]; ok && scopesStr != "" {
				tool["scopes"] = strings.Split(scopesStr, ",")
			}
			
			tools = append(tools, tool)
		}
	}
	
	enc := json.NewEncoder(writer)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]interface{}{
		"tools": tools,
		"count": len(tools),
	})
}
