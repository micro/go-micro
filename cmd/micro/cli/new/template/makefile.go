package template

var Makefile = `.PHONY: proto build run test clean docker

# Generate protobuf files
proto:
	protoc --proto_path=. --micro_out=. --go_out=. proto/*.proto

# Build the service
build:
	go build -o bin/{{.Alias}} .

# Run the service
run:
	go run .

# Run with hot reload (requires air: go install github.com/air-verse/air@latest)
dev:
	air

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# List MCP tools exposed by this service
mcp-tools:
	micro mcp list

# Test an MCP tool interactively
mcp-test:
	micro mcp test

# Start MCP server for Claude Code
mcp-serve:
	micro mcp serve

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html

# Build Docker image
docker:
	docker build -t {{.Alias}}:latest .

# Lint code
lint:
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Update dependencies
deps:
	go mod tidy
	go mod download
`
