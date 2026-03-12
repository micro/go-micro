package template

// Simple templates generate a service using plain Go structs and JSON encoding.
// No protobuf, no code generation — just Go.

var SimpleMain = `package main

import (
	"context"
	"fmt"
	"log"

	"go-micro.dev/v5"
)

// Request is the input for the greeting.
type Request struct {
	Name string ` + "`" + `json:"name"` + "`" + `
}

// Response is the greeting result.
type Response struct {
	Message string ` + "`" + `json:"message"` + "`" + `
}

// {{title .Alias}} is the service handler.
type {{title .Alias}} struct{}

// Call greets a person by name.
func (h *{{title .Alias}}) Call(ctx context.Context, req *Request, rsp *Response) error {
	rsp.Message = "Hello " + req.Name
	return nil
}

func main() {
	service := micro.New("{{lower .Alias}}")

	service.Init()

	if err := service.Handle(new({{title .Alias}})); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Starting {{lower .Alias}} service on :0 (random port)")
	fmt.Println()
	fmt.Println("Or set a fixed address:")
	fmt.Println("  service := micro.New(\"{{lower .Alias}}\", micro.Address(\":8080\"))")

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
`

var SimpleMainMCP = `package main

import (
	"context"
	"fmt"
	"log"

	"go-micro.dev/v5"
	"go-micro.dev/v5/gateway/mcp"
)

// Request is the input for the greeting.
type Request struct {
	Name string ` + "`" + `json:"name"` + "`" + `
}

// Response is the greeting result.
type Response struct {
	Message string ` + "`" + `json:"message"` + "`" + `
}

// {{title .Alias}} is the service handler.
type {{title .Alias}} struct{}

// Call greets a person by name and returns a welcome message.
//
// @example {"name": "Alice"}
func (h *{{title .Alias}}) Call(ctx context.Context, req *Request, rsp *Response) error {
	rsp.Message = "Hello " + req.Name
	return nil
}

func main() {
	service := micro.New("{{lower .Alias}}",
		micro.Address(":9090"),
		mcp.WithMCP(":3001"),
	)

	service.Init()

	if err := service.Handle(new({{title .Alias}})); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Starting {{lower .Alias}} service")
	fmt.Println()
	fmt.Println("  Service:   http://localhost:9090")
	fmt.Println("  MCP Tools: http://localhost:3001/mcp/tools")
	fmt.Println()
	fmt.Println("Use with Claude Code:")
	fmt.Println("  micro mcp serve")

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
`

var SimpleMakefile = `.PHONY: build run test clean lint fmt

# Build the service
build:
	go build -o bin/{{.Alias}} .

# Run the service
run:
	go run .

# Run with micro (gateway + hot reload)
dev:
	micro run

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html

# Lint code
lint:
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...
`

var SimpleModule = `module {{.Dir}}

go 1.22

require go-micro.dev/v5 latest
`

var SimpleReadme = `# {{title .Alias}} Service

Generated with ` + "`" + `micro new --simple {{.Alias}}` + "`" + `

## Getting Started

Run the service:

` + "```bash" + `
go run .
` + "```" + `

Call it:

` + "```bash" + `
curl -XPOST \
  -H 'Content-Type: application/json' \
  -H 'Micro-Endpoint: {{title .Alias}}.Call' \
  -d '{"name": "Alice"}' \
  http://localhost:9090
` + "```" + `

## Development

` + "```bash" + `
make run     # Run the service
make test    # Run tests
make build   # Build binary
micro run    # Run with gateway + hot reload
` + "```" + `
`

var SimpleReadmeMCP = `# {{title .Alias}} Service

Generated with ` + "`" + `micro new {{.Alias}}` + "`" + `

## Getting Started

Run the service:

` + "```bash" + `
go run .
` + "```" + `

Call it:

` + "```bash" + `
curl -XPOST \
  -H 'Content-Type: application/json' \
  -H 'Micro-Endpoint: {{title .Alias}}.Call' \
  -d '{"name": "Alice"}' \
  http://localhost:9090
` + "```" + `

## MCP & AI Agents

This service is MCP-enabled. When running, AI agents can discover
and call your endpoints automatically.

**MCP tools:** http://localhost:3001/mcp/tools

### Use with Claude Code

` + "```bash" + `
micro mcp serve
` + "```" + `

## Development

` + "```bash" + `
make run     # Run the service
make test    # Run tests
make build   # Build binary
micro run    # Run with gateway + hot reload
` + "```" + `
`
