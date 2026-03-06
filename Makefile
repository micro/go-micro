NAME = micro
GIT_COMMIT = $(shell git rev-parse --short HEAD)
GIT_TAG = $(shell git describe --abbrev=0 --tags --always --match "v*")
GIT_IMPORT = go-micro.dev/v5/cmd/micro
BUILD_DATE = $(shell date +%s)
LDFLAGS = -X $(GIT_IMPORT).BuildDate=$(BUILD_DATE) -X $(GIT_IMPORT).GitCommit=$(GIT_COMMIT) -X $(GIT_IMPORT).GitTag=$(GIT_TAG)

# GORELEASER_DOCKER_IMAGE = ghcr.io/goreleaser/goreleaser-cross:v1.25.7
GORELEASER_DOCKER_IMAGE = ghcr.io/goreleaser/goreleaser:latest

.PHONY: test test-race test-coverage lint fmt install-tools proto clean help gorelease-dry-run gorelease-dry-run-docker

# Default target
help:
	@echo "Go Micro Development Tasks"
	@echo ""
	@echo "  make test          - Run tests"
	@echo "  make test-race     - Run tests with race detector"
	@echo "  make test-coverage - Run tests with coverage"
	@echo "  make lint          - Run linter"
	@echo "  make fmt           - Format code"
	@echo "  make install-tools - Install development tools"
	@echo "  make proto         - Generate protobuf code"
	@echo "  make clean         - Clean build artifacts"

$(NAME):
	CGO_ENABLED=0 go build -ldflags "-s -w ${LDFLAGS}" -o $(NAME) cmd/micro/main.go

# Run tests
test:
	go test -v ./...

# Run tests with race detector
test-race:
	go test -v -race ./...

# Run tests with coverage
test-coverage:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	gofmt -s -w .
	goimports -w .

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/kyoh86/richgo@latest
	go install go-micro.dev/v5/cmd/protoc-gen-micro@latest
	@echo "Tools installed successfully"

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	find . -name "*.proto" -not -path "./vendor/*" -exec protoc --proto_path=. --micro_out=. --go_out=. {} \;

# Clean build artifacts
clean:
	rm -f coverage.out coverage.html
	find . -name "*.test" -type f -delete
	go clean -cache -testcache

# Try binary release
gorelease-dry-run:
	docker run \
		--rm \
		-e CGO_ENABLED=0 \
		-v $(CURDIR):/$(NAME) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /$(NAME) \
		$(GORELEASER_DOCKER_IMAGE) \
		--clean --verbose --skip=publish,validate --snapshot

